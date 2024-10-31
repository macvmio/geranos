package layout

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/macvmio/geranos/pkg/dirimage"
	"github.com/macvmio/geranos/pkg/duplicator"
	"github.com/macvmio/geranos/pkg/sketch"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const OSWindows = "windows"

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
	if runtime.GOOS == OSWindows {
		refStr = strings.ReplaceAll(refStr, ":", "@")
	}
	return filepath.Join(lm.rootDir, refStr)
}

func (lm *Mapper) dirToRef(dir string) (name.Reference, error) {
	processedPath := strings.TrimPrefix(filepath.Clean(dir), lm.rootDir)
	processedPath = filepath.ToSlash(strings.Trim(processedPath, "/\\"))
	if runtime.GOOS == OSWindows {
		processedPath = strings.Replace(processedPath, "@", ":", -1)
	}
	return name.ParseReference(processedPath, name.StrictValidation)
}

func (lm *Mapper) WriteIfNotPresent(ctx context.Context, img v1.Image, ref name.Reference) error {
	originalDigest, err := img.Digest()
	if err != nil {
		return fmt.Errorf("failed to read origin manifest: %w", err)
	}
	localImg, err := dirimage.Read(ctx, lm.refToDir(ref), dirimage.WithOmitLayersContent())
	if err == nil {
		localDigest, err := localImg.Digest()
		if err == nil && localDigest == originalDigest {
			fmt.Println("skipped writing because digests are the same")
			return nil
		}
	}
	return lm.Write(ctx, img, ref)
}

func (lm *Mapper) Write(ctx context.Context, img v1.Image, ref name.Reference) error {
	if img == nil {
		return errors.New("nil image provided")
	}
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
		st := Statistics{}
		st.SourceBytesCount.Store(layer.Size)
		lm.stats.Add(&st)
	}

	bytesClonedCount, matchedSegmentsCount, err := lm.sketcher.Sketch(destinationDir, *manifest)
	if err != nil {
		// TODO: ensure we don't delete anything useful _ = os.RemoveAll(destinationDir)
		return err
	}
	st := Statistics{}
	st.BytesClonedCount.Store(bytesClonedCount)
	st.MatchedSegmentsCount.Store(matchedSegmentsCount)
	lm.stats.Add(&st)

	convertedImage, err := dirimage.Convert(img)
	if err != nil {
		return fmt.Errorf("unable to convert to dirimage: %w", err)
	}
	err = convertedImage.Write(ctx, destinationDir, lm.opts...)
	if err != nil {
		return fmt.Errorf("unable to write dirimage to '%v': %w", destinationDir, err)
	}

	st = Statistics{}
	st.BytesWrittenCount.Store(convertedImage.BytesWrittenCount.Load())
	st.BytesSkippedCount.Store(convertedImage.BytesSkippedCount.Load())
	st.BytesReadCount.Store(convertedImage.BytesReadCount.Load())
	lm.stats.Add(&st)
	return nil
}

func (lm *Mapper) Rehash(ctx context.Context, ref name.Reference) error {
	refStr := lm.refToDir(ref)
	img, err := dirimage.Read(ctx, refStr, lm.opts...)
	if err != nil {
		return fmt.Errorf("unable to read dirimage: %w", err)
	}
	st := Statistics{}
	st.BytesReadCount.Store(img.BytesReadCount.Load())
	lm.stats.Add(&st)
	return img.WriteConfigAndManifest(refStr)
}

func (lm *Mapper) Read(ctx context.Context, ref name.Reference) (v1.Image, error) {
	refStr := lm.refToDir(ref)
	img, err := dirimage.Read(ctx, refStr, lm.opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to read dirimage: %w", err)
	}
	st := Statistics{}
	st.BytesReadCount.Store(img.BytesReadCount.Load())
	lm.stats.Add(&st)
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
	Ref         name.Reference
	DiskUsage   string
	Size        int64
	HasManifest bool
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

func (lm *Mapper) containsManifest(ref name.Reference) bool {
	_, err := dirimage.Read(context.Background(), lm.refToDir(ref), dirimage.WithOmitLayersContent())
	return err == nil
}

func (lm *Mapper) List() ([]Properties, error) {
	res := make([]Properties, 0)
	err := filepath.WalkDir(lm.rootDir, func(path string, d fs.DirEntry, argErr error) error {
		if d == nil || !d.IsDir() {
			return nil
		}

		ref, err := lm.dirToRef(path)
		if err != nil {
			// fmt.Printf("skipping error %v\n", err)
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
			Ref:         ref,
			DiskUsage:   diskUsage,
			Size:        dirSize,
			HasManifest: lm.containsManifest(ref),
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
	return os.RemoveAll(lm.refToDir(ref))
}

func (lm *Mapper) Stats() ImmutableStatistics {
	return ImmutableStatistics{
		SourceBytesCount:     lm.stats.SourceBytesCount.Load(),
		BytesWrittenCount:    lm.stats.BytesWrittenCount.Load(),
		BytesSkippedCount:    lm.stats.BytesSkippedCount.Load(),
		BytesReadCount:       lm.stats.BytesReadCount.Load(),
		BytesClonedCount:     lm.stats.BytesClonedCount.Load(),
		CompressedBytesCount: lm.stats.CompressedBytesCount.Load(),
		MatchedSegmentsCount: lm.stats.MatchedSegmentsCount.Load(),
	}
}
