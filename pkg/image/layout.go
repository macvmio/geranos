package image

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/tomekjarosik/geranos/pkg/image/dirimage"
	"github.com/tomekjarosik/geranos/pkg/image/duplicator"
	"github.com/tomekjarosik/geranos/pkg/image/filesegment"
	"github.com/tomekjarosik/geranos/pkg/image/sketch"
	"github.com/tomekjarosik/geranos/pkg/image/sparsefile"
	"golang.org/x/sync/errgroup"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const LocalManifestFilename = ".oci.manifest.json"
const ConfigMediaType = types.MediaType("application/online.jarosik.tomasz.v1.config+json")

type LayoutMapper struct {
	rootDir  string
	sketcher *sketch.Sketcher

	opts  *options
	stats Statistics
}

type Layout struct {
	Filenames []string
	Digests   map[string][]v1.Hash
	Sizes     []int64
}

func NewLayoutMapper(rootDir string, opt ...Option) *LayoutMapper {
	return &LayoutMapper{
		rootDir:  rootDir,
		sketcher: sketch.NewSketcher(rootDir, LocalManifestFilename),
		opts:     makeOptions(opt...),
	}
}

func (lm *LayoutMapper) refToDir(ref name.Reference) string {
	return filepath.Join(lm.rootDir, ref.String())
}

func (lm *LayoutMapper) contentMatches(destinationDir string, segment *filesegment.Descriptor) bool {
	fname := filepath.Join(destinationDir, segment.Filename())
	f, err := os.OpenFile(fname, os.O_RDONLY, 0666)
	if err != nil {
		return false
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			lm.opts.printf("error while closing file %v, got %v", segment.Filename(), err)
		}
	}(f)
	l, err := filesegment.NewLayer(fname, filesegment.WithRange(segment.Start(), segment.Stop()))
	if err != nil {
		return false
	}
	d, err := l.Digest()
	lm.stats.Add(&Statistics{
		BytesReadCount: segment.Length(),
	})
	if err != nil {
		return false
	}
	if d == segment.Digest() {
		return true
	}
	return false
}

func (lm *LayoutMapper) writeToSegment(destinationDir string, segment *filesegment.Descriptor, src io.ReadCloser) (written int64, skipped int64, err error) {
	// Here: we have io.ReadCloser dumping to a file at given location
	f, err := filesegment.NewWriter(destinationDir, segment)
	if err != nil {
		return 0, 0, err
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			lm.opts.printf("error while closing file %v, got %v", segment.Filename(), err)
		}
	}(f)

	written, skipped, err = sparsefile.Copy(f, src)
	if written+skipped != segment.Length() {
		return written, skipped, fmt.Errorf("invalid numer of bytes written+skipped: segment length: %d, written+skipped: %d", segment.Length(), written+skipped)
	}
	return written, skipped, err
}

func (lm *LayoutMapper) writeLayer(destinationDir string, segment *filesegment.Descriptor, layer v1.Layer) (written int64, skipped int64, err error) {
	if layer == nil {
		return 0, 0, errors.New("nil layer provided")
	}

	rc, err := layer.Uncompressed()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to access uncompressed layer: %w", err)
	}
	defer rc.Close()
	return lm.writeToSegment(destinationDir, segment, rc)
}

func (lm *LayoutMapper) Write(ctx context.Context, img v1.Image, ref name.Reference) error {
	destinationDir := lm.refToDir(ref)
	err := os.MkdirAll(destinationDir, 0o777)
	if err != nil {
		return err
	}

	manifest, err := img.Manifest()
	if err != nil {
		return err
	}

	bytesClonedCount, matchedSegmentsCount, err := lm.sketcher.Sketch(destinationDir, *manifest)
	if err != nil {
		// TODO: ensure we don't delete anything useful _ = os.RemoveAll(destinationDir)
		return err
	}
	lm.stats.Add(&Statistics{BytesClonedCount: bytesClonedCount, MatchedSegmentsCount: matchedSegmentsCount})

	type Job struct {
		Descriptor filesegment.Descriptor
		Layer      v1.Layer
	}
	type JobResult struct {
		Job Job
		err error
	}
	workersCount := min(lm.opts.workersCount, runtime.NumCPU())
	jobs := make(chan Job, workersCount)
	results := make(chan JobResult, workersCount)

	g, ctx := errgroup.WithContext(ctx)

	for w := 0; w < workersCount; w++ {
		g.Go(func() error {
			for job := range jobs {
				var jobErr error
				for i := 0; i < lm.opts.networkFailureRetryCount; i++ {
					if lm.contentMatches(destinationDir, &job.Descriptor) {
						break
					}
					written, skipped, err := lm.writeLayer(destinationDir, &job.Descriptor, job.Layer)
					lm.opts.printf("written=%d, skipped=%d\n", written, skipped)
					lm.stats.Add(&Statistics{
						BytesWrittenCount: written,
						BytesSkippedCount: skipped,
					})
					if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
						continue
					}
					jobErr = err
					break
				}
				results <- JobResult{Job: job, err: jobErr}
			}
			return nil
		})
	}

	g.Go(func() error {
		defer close(jobs)
		for _, l := range manifest.Layers {
			d, err := filesegment.ParseDescriptor(l)
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err() // Early return on context cancellation.
			default:
				l, err := img.LayerByDigest(d.Digest())
				if err != nil {
					lm.opts.printf("invalid seg.Digest")
					l = nil
				}
				jobs <- Job{Descriptor: *d, Layer: l}
			}
		}
		return nil
	})
	go func() {
		err = g.Wait()
		if err != nil {
			fmt.Println(err)
		}
		close(results)
	}()

	for res := range results {
		if res.err != nil {
			return fmt.Errorf("failed writing to file '%v' at offset '%v': %w", res.Job.Descriptor.Filename(), res.Job.Descriptor.Start(), res.err)
		}
	}
	rawManifest, err := img.RawManifest()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destinationDir, LocalManifestFilename), rawManifest, 0o777)
}

func (lm *LayoutMapper) Read(ctx context.Context, ref name.Reference) (v1.Image, error) {

	img, err := dirimage.Read(ctx, filepath.Join(lm.rootDir, ref.String()), dirimage.WithChunkSize(lm.opts.chunkSize))
	if err != nil {
		return nil, fmt.Errorf("unable to read dirimage: %w", err)
	}
	lm.stats.Add(&Statistics{BytesReadCount: img.BytesReadCount})
	return img, err
}

// IsDirWithOnlyFiles checks if the given path is a directory that contains only files (no subdirectories).
func IsDirWithOnlyFiles(path string) (bool, error) {
	// Check the provided path is indeed a directory
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if !fileInfo.IsDir() {
		return false, nil // Not even a directory
	}

	// Read the directory contents
	contents, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	// Iterate over the directory contents
	for _, content := range contents {
		if content.IsDir() {
			return false, nil // Found a subdirectory, so return false
		}
	}

	return true, nil // No subdirectories found, only files
}

func (lm *LayoutMapper) Adopt(src string, ref name.Reference, failIfContainsSubdirectories bool) error {
	isFlatDir, err := IsDirWithOnlyFiles(src)
	if err != nil {
		return fmt.Errorf("unable to verify if directory is flat: %w", err)
	}
	if !isFlatDir && failIfContainsSubdirectories {
		return fmt.Errorf("directory with subdirectories are not supported")
	}
	if failIfContainsSubdirectories {
		lm.opts.printf("warning: subdirectories will be ignored")
	}
	return duplicator.CloneDirectory(src, lm.refToDir(ref), false)
}

type Properties struct {
	Ref       name.Reference
	DiskUsage string
	Size      int64
}

func directorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}

func (lm *LayoutMapper) ContainsManifest(ref name.Reference) bool {
	return true
}

func (lm *LayoutMapper) List() ([]Properties, error) {
	res := make([]Properties, 0)
	err := filepath.WalkDir(lm.rootDir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}
		processedPath := strings.TrimPrefix(path, lm.rootDir)
		processedPath = strings.Trim(processedPath, "/")
		ref, err := name.ParseReference(processedPath, name.StrictValidation)
		if err != nil {
			return nil
		}
		if !lm.ContainsManifest(ref) {
			return nil
		}
		dirSize, err := directorySize(path)
		if err != nil {
			dirSize = -1
		}
		diskUsage, err := directoryDiskUsage(path)
		if err != nil {
			return err
		}
		res = append(res, Properties{
			Ref:       ref,
			DiskUsage: diskUsage,
			Size:      dirSize,
		})
		return nil
	})
	return res, err
}

func (lm *LayoutMapper) Clone(src name.Reference, dst name.Reference) error {
	return duplicator.CloneDirectory(lm.refToDir(src), lm.refToDir(dst), true)
}

func (lm *LayoutMapper) Remove(src name.Reference) error {
	ref, err := name.ParseReference(src.String(), name.StrictValidation)
	if err != nil {
		return fmt.Errorf("unable to valid reference: %w", err)
	}
	return os.RemoveAll(filepath.Join(lm.rootDir, lm.refToDir(ref)))
}

func (lm *LayoutMapper) Stats() Statistics {
	return lm.stats
}
