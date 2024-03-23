package image

import (
	"errors"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/tomekjarosik/geranos/pkg/image/segmentlayer"
	"github.com/tomekjarosik/geranos/pkg/image/sparsefile"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const LocalManifestFilename = ".oci.manifest.json"
const ConfigMediaType = types.MediaType("application/online.jarosik.tomasz.v1.config+json")
const FilenameAnnotationKey = "filename"
const RangeAnnotationKey = "range"

type LayoutMapper struct {
	rootDir              string
	makeshiftConstructor SketchConstructor
}

type Layout struct {
	Filenames []string
	Digests   map[string][]v1.Hash
	Sizes     []int64
}

type SketchConstructor interface {
	Construct(dir string, fileRecipes []*FileRecipe) error
}

type noopMakeshiftConstructor struct {
}

func (mc *noopMakeshiftConstructor) Construct(dir string, fileRecipes []*FileRecipe) error {
	return nil
}

func NewLayoutMapper(rootDir string) *LayoutMapper {
	return &LayoutMapper{
		rootDir:              rootDir,
		makeshiftConstructor: &noopMakeshiftConstructor{},
	}
}

func (lm *LayoutMapper) writeLayer(destinationDir string, segment *FileSegmentRecipe, layer v1.Layer) (written int64, skipped int64, err error) {
	if layer == nil {
		return 0, 0, errors.New("nil layer provided")
	}
	f, err := os.OpenFile(filepath.Join(destinationDir, segment.Filename), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return 0, 0, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("error while closing file %v, got %v", segment.Filename, err)
		}
	}(f)

	_, err = f.Seek(segment.Start, io.SeekStart)
	if err != nil {
		return 0, 0, err
	}
	rc, err := layer.Uncompressed()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to access uncompressed layer: %w", err)
	}
	defer rc.Close()

	return sparsefile.Copy(f, rc)
}

func (lm *LayoutMapper) Write(img v1.Image, ref name.Reference, progress chan ProgressUpdate) error {
	recipes, err := CreateFileRecipesFromImage(img)
	if err != nil {
		return err
	}
	destinationDir := filepath.Join(lm.rootDir, ref.String())
	err = os.MkdirAll(destinationDir, 0o777)
	if err != nil {
		return err
	}
	err = lm.makeshiftConstructor.Construct(destinationDir, recipes)
	if err != nil {
		// TODO: ensure we don't delete anything useful _ = os.RemoveAll(destinationDir)
		return err
	}
	type Job struct {
		Recipe FileSegmentRecipe
		Layer  v1.Layer
	}
	type JobResult struct {
		Job Job
		err error
	}
	var wg sync.WaitGroup
	workersCount := min(8, runtime.NumCPU()) // TODO: make it configurable
	jobs := make(chan Job, workersCount)
	results := make(chan JobResult, workersCount)

	for w := 0; w < workersCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				written, skipped, err := lm.writeLayer(destinationDir, &job.Recipe, job.Layer)
				log.Printf("written=%d, skipped=%d\n", written, skipped)
				// TODO: Retry here up to configurable number of times
				results <- JobResult{Job: job, err: err}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		for _, r := range recipes {
			for _, seg := range r.Segments {
				l, err := img.LayerByDigest(seg.Digest)
				if err != nil {
					log.Printf("invalid seg.Digest")
					l = nil
				}
				jobs <- Job{Recipe: seg, Layer: l}
			}
		}
		close(jobs)
	}()
	for res := range results {
		if res.err != nil {
			return fmt.Errorf("failed writing to file '%v' at offset '%v': %w", res.Job.Recipe.Filename, res.Job.Recipe.Start, err)
		}
	}
	rawManifest, err := img.RawManifest()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destinationDir, LocalManifestFilename), rawManifest, 0o777)
}

func (lm *LayoutMapper) splitToLayers(fullpath string, chunkSize int64, workersCount int) ([]*segmentlayer.Layer, error) {
	f, err := os.Stat(fullpath)
	if err != nil {
		return nil, err
	}
	if f.Size() < chunkSize {
		l, err := segmentlayer.FromFile(fullpath)
		if err != nil {
			return nil, err
		}
		return []*segmentlayer.Layer{l}, nil
	}

	type layerResult struct {
		layer *segmentlayer.Layer
		err   error
	}

	maxIdx := f.Size() - 1
	jobs := make(chan int64, workersCount)
	results := make(chan layerResult, workersCount)

	var wg sync.WaitGroup
	for w := 0; w < workersCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for start := range jobs {
				stop := start + chunkSize - 1
				if stop > maxIdx {
					stop = maxIdx
				}
				l, err := segmentlayer.FromFile(fullpath, segmentlayer.WithRange(start, stop))
				results <- layerResult{
					layer: l,
					err:   err,
				}
				if err == nil {
					// Precompute hashes concurrently
					_, _ = l.DiffID()
					_, _ = l.Digest()
				}
			}
		}()
	}
	go func() {
		for start := int64(0); start <= maxIdx; start += chunkSize {
			jobs <- start
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()
	sortedLayers := make([]layerResult, 0)
	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		sortedLayers = append(sortedLayers, r)
	}
	sort.Slice(sortedLayers, func(i, j int) bool {
		return sortedLayers[i].layer.Start() < sortedLayers[j].layer.Start()
	})
	res := make([]*segmentlayer.Layer, 0)
	for _, r := range sortedLayers {
		res = append(res, r.layer)
	}
	return res, nil
}

func (lm *LayoutMapper) Read(ref name.Reference, opt ...Option) (v1.Image, error) {
	img := empty.Image
	img = mutate.ConfigMediaType(img, ConfigMediaType)
	opts := makeOptions(opt...)
	dirEntries, err := os.ReadDir(filepath.Join(lm.rootDir, ref.String()))
	if err != nil {
		return nil, err
	}
	for _, entry := range dirEntries {
		if entry.IsDir() {
			log.Printf("unexpected subdirectory '%v', skipping", entry.Name())
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			log.Printf("skipping file '%v' because it starts with a dot", entry.Name())
			continue
		}
		layers, err := lm.splitToLayers(filepath.Join(lm.rootDir, ref.String(), entry.Name()), opts.chunkSize, opts.workersCount)
		if err != nil {
			return nil, err
		}
		addendums := make([]mutate.Addendum, 0)
		for _, l := range layers {
			addendums = append(addendums, mutate.Addendum{
				Layer:   l,
				History: v1.History{},
				Annotations: map[string]string{
					FilenameAnnotationKey: entry.Name(),
					RangeAnnotationKey:    fmt.Sprintf("%d-%d", l.Start(), l.Stop()),
				},
				MediaType: segmentlayer.FileSegmentMediaType,
			})
		}
		img, err = mutate.Append(img, addendums...)
		if err != nil {
			return nil, fmt.Errorf("unable to append layers to image: %w", err)
		}
	}
	diffIDs := make([]v1.Hash, 0)
	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}
	for _, l := range layers {
		h, err := l.DiffID()
		if err != nil {
			return nil, err
		}
		diffIDs = append(diffIDs, h)
	}
	return mutate.ConfigFile(img, &v1.ConfigFile{
		Architecture: "arm64", // TODO
		Author:       "TODO",
		Container:    "geranos", // TODO: Rename whole package to 'Geranos'
		Created:      v1.Time{Time: time.Now()},
		OS:           "",
		RootFS:       v1.RootFS{Type: "layers", DiffIDs: diffIDs},
		OSVersion:    "TODO",
		Variant:      "",
	})
}
