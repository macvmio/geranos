package layout

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/mobileinf/geranos/pkg/dirimage"
	"github.com/mobileinf/geranos/pkg/filesegment"
	"runtime"

	"github.com/mobileinf/geranos/pkg/duplicator"

	"github.com/mobileinf/geranos/pkg/sketch"
	"github.com/mobileinf/geranos/pkg/sparsefile"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const ConfigMediaType = types.MediaType("application/online.jarosik.tomasz.v1.config+json")

type Mapper struct {
	rootDir  string
	sketcher *sketch.Sketcher

	opts  []dirimage.Option
	stats Statistics
}

type Layout struct {
	Filenames []string
	Digests   map[string][]v1.Hash
	Sizes     []int64
}

func NewMapper(rootDir string, opts ...dirimage.Option) *Mapper {
	return &Mapper{
		rootDir:  rootDir,
		sketcher: sketch.NewSketcher(rootDir, dirimage.LocalManifestFilename),
		opts:     opts,
	}
}

func (lm *Mapper) refToDir(ref name.Reference) string {
	refStr := ref.String()
	if runtime.GOOS == "windows" {
		refStr = strings.ReplaceAll(refStr, ":", "@")
	}
	return filepath.Join(lm.rootDir, refStr)
}

func (lm *Mapper) writeToSegment(destinationDir string, segment *filesegment.Descriptor, src io.ReadCloser) (written int64, skipped int64, err error) {
	f, err := filesegment.NewWriter(destinationDir, segment)
	if err != nil {
		return 0, 0, err
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			fmt.Printf("error while closing file %v, got %v", segment.Filename(), err)
		}
	}(f)

	written, skipped, err = sparsefile.Copy(f, src)
	if written+skipped != segment.Length() {
		return written, skipped, fmt.Errorf("invalid numer of bytes written+skipped: segment length: %d, written+skipped: %d", segment.Length(), written+skipped)
	}
	return written, skipped, err
}

func (lm *Mapper) writeLayer(destinationDir string, segment *filesegment.Descriptor, layer v1.Layer) (written int64, skipped int64, err error) {
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

func (lm *Mapper) Write(ctx context.Context, img v1.Image, ref name.Reference) error {
	destinationDir := lm.refToDir(ref)
	err := os.MkdirAll(destinationDir, 0o777)
	if err != nil {
		return fmt.Errorf("unable to create directory for writing: %w", err)
	}

	manifest, err := img.Manifest()
	if err != nil {
		return err
	}
	for _, layer := range manifest.Layers {
		lm.stats.Add(&Statistics{SourceBytesCount: layer.Size})
	}

	bytesClonedCount, matchedSegmentsCount, err := lm.sketcher.Sketch(destinationDir, *manifest)
	if err != nil {
		// TODO: ensure we don't delete anything useful _ = os.RemoveAll(destinationDir)
		return err
	}
	lm.stats.Add(&Statistics{BytesClonedCount: bytesClonedCount, MatchedSegmentsCount: matchedSegmentsCount})

	convertedImage, err := dirimage.Convert(img)
	if err != nil {
		return fmt.Errorf("unable to convert to dirimage: %w", err)
	}
	err = convertedImage.Write(ctx, destinationDir, lm.opts...)
	if err != nil {
		return fmt.Errorf("unable to write dirimage to '%v': %w", destinationDir, err)
	}
	lm.stats.Add(&Statistics{
		BytesWrittenCount: convertedImage.BytesWrittenCount,
		BytesSkippedCount: convertedImage.BytesSkippedCount,
		BytesReadCount:    convertedImage.BytesReadCount,
	})
	return nil
}

func (lm *Mapper) Read(ctx context.Context, ref name.Reference) (v1.Image, error) {
	refStr := lm.refToDir(ref)
	img, err := dirimage.Read(ctx, refStr, lm.opts...)
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

func (lm *Mapper) Adopt(src string, ref name.Reference, failIfContainsSubdirectories bool) error {
	isFlatDir, err := IsDirWithOnlyFiles(src)
	if err != nil {
		return fmt.Errorf("unable to verify if directory is flat: %w", err)
	}
	if !isFlatDir && failIfContainsSubdirectories {
		return fmt.Errorf("directory with subdirectories are not supported")
	}
	if failIfContainsSubdirectories {
		fmt.Printf("warning: subdirectories will be ignored")
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

func (lm *Mapper) ContainsManifest(ref name.Reference) bool {
	return true
}

// ContainsAny returns true if there is a directory corresponding to the provided reference
// It does not validate if that directory contains anything useful
func (lm *Mapper) ContainsAny(ref name.Reference) (bool, error) {
	info, err := os.Stat(lm.refToDir(ref))
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (lm *Mapper) List() ([]Properties, error) {
	res := make([]Properties, 0)
	err := filepath.WalkDir(lm.rootDir, func(path string, d fs.DirEntry, err error) error {
		if d == nil || !d.IsDir() {
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
		diskUsage, err := DirectoryDiskUsage(path)
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

func (lm *Mapper) Clone(src name.Reference, dst name.Reference) error {
	return duplicator.CloneDirectory(lm.refToDir(src), lm.refToDir(dst), true)
}

func (lm *Mapper) Remove(src name.Reference) error {
	ref, err := name.ParseReference(src.String(), name.StrictValidation)
	if err != nil {
		return fmt.Errorf("unable to valid reference: %w", err)
	}
	return os.RemoveAll(filepath.Join(lm.rootDir, lm.refToDir(ref)))
}

func (lm *Mapper) Stats() Statistics {
	return lm.stats
}
