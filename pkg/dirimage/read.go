package dirimage

import (
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/macvmio/geranos/pkg/filesegment"
	"golang.org/x/sync/errgroup"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

type hasAnnotations interface {
	Annotations() map[string]string
}

type hasLength interface {
	Length() int64
}

func precomputeHashes(ctx context.Context, layers []v1.Layer, workersCount int) (bytesReadCount int64, err error) {
	jobs := make(chan v1.Layer, workersCount)
	g, ctx := errgroup.WithContext(ctx)

	var aBytesReadCount atomic.Int64
	for w := 0; w < workersCount; w++ {
		g.Go(func() error {
			for l := range jobs {
				_, _ = l.DiffID()
				_, _ = l.Digest()
				hl, ok := l.(hasLength)
				if !ok {
					return fmt.Errorf("layer does not implement Length() method")
				}
				aBytesReadCount.Add(2 * hl.Length())
			}
			return nil
		})
	}
	g.Go(func() error {
		defer close(jobs)
		for _, l := range layers {
			select {
			case jobs <- l: // this blocks here waiting for a worker to pick up a job
			case <-ctx.Done():
				return ctx.Err() // Exit if the context is canceled
			}
		}
		return nil
	})

	err = g.Wait()
	return aBytesReadCount.Load(), err
}

func prepareLayers(dir string, cfgFile *v1.ConfigFile, opts *options) ([]v1.Layer, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory: '%v': %w", dir, err)
	}

	layers := make([]v1.Layer, 0)
	if opts.omitLayersContent {
		// Use the RootFS from the config file
		if cfgFile.RootFS.Type == "" || len(cfgFile.RootFS.DiffIDs) == 0 {
			return nil, fmt.Errorf("config file must contain RootFS when omitting layer content")
		}
		// Read manifest and config files to reconstruct layers without content
		return prepareLayersFromManifestAndConfig(dir, cfgFile)
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			opts.printf("unexpected subdirectory '%v', skipping", entry.Name())
			continue
		}
		if strings.HasPrefix(entry.Name(), ".") {
			opts.printf("skipping file '%v' because it starts with a dot", entry.Name())
			continue
		}

		fileLayers, err := filesegment.Split(filepath.Join(dir, entry.Name()), opts.chunkSize, filesegment.WithLogFunction(opts.printf))
		if err != nil {
			return nil, err
		}
		// Append each *filesegment.Layer to the []v1.Layer slice
		for _, fl := range fileLayers {
			layers = append(layers, fl)
		}
	}
	return layers, nil
}

func prepareAddendums(layers []v1.Layer) ([]mutate.Addendum, error) {
	addendums := make([]mutate.Addendum, 0)
	for _, l := range layers {
		mt, err := l.MediaType()
		if err != nil {
			return nil, err
		}
		la, ok := l.(hasAnnotations)
		if !ok {
			return nil, fmt.Errorf("layer does not implement Annotations() method")
		}
		addendums = append(addendums, mutate.Addendum{
			Layer:       l,
			History:     v1.History{},
			Annotations: la.Annotations(),
			MediaType:   mt,
		})
	}
	return addendums, nil
}

func prepareDiffIDs(layers []v1.Layer) ([]v1.Hash, error) {
	diffIDs := make([]v1.Hash, 0)
	for _, l := range layers {
		diffID, err := l.DiffID()
		if err != nil {
			return nil, err
		}
		diffIDs = append(diffIDs, diffID)
	}
	return diffIDs, nil
}

func prepareImage(cfg *v1.ConfigFile, addendums []mutate.Addendum) (v1.Image, error) {
	img := empty.Image
	img = mutate.MediaType(img, ManifestMediaType)
	img = mutate.ConfigMediaType(img, ConfigMediaType)
	img, err := mutate.Append(img, addendums...)
	if err != nil {
		return nil, fmt.Errorf("unable to append layers to image: %w", err)
	}
	img, err = mutate.ConfigFile(img, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to mutate config file: %w", err)
	}
	return img, nil
}

func prepareConfigFile(dir string, requireFile bool) (*v1.ConfigFile, error) {
	configFilePath := filepath.Join(dir, LocalConfigFilename)

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			if requireFile {
				return nil, fmt.Errorf("config file is required when skipVerification is true")
			}
			// File does not exist, return a new config
			return &v1.ConfigFile{
				Container: "geranos",
				Created:   v1.Time{Time: time.Now()},
				Config: v1.Config{
					Labels: map[string]string{
						"org.opencontainers.image.title":       "geranos",
						"org.opencontainers.image.description": "default description of the image",
						"org.opencontainers.image.authors":     "macvmio",
						"org.opencontainers.image.url":         "https://github.com/macvmio/geranos",
						"org.opencontainers.image.source":      "https://github.com/macvmio/geranos",
						//"org.opencontainers.image.version":     "", // Replace with your actual version
						"org.opencontainers.image.created":  time.Now().Format(time.RFC3339),
						"org.opencontainers.image.licenses": "Apache-2.0", // Replace with your license
					},
				},
			}, nil
		}
		// Some other error occurred when reading the file
		return nil, fmt.Errorf("unable to read config file: %w", err)
	}

	// File exists, unmarshal the JSON content
	cfg := &v1.ConfigFile{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("unable to parse config file: %w", err)
	}
	return cfg, nil
}

func computeRootFS(ctx context.Context, layers []v1.Layer, opts *options) (v1.RootFS, int64, error) {
	bytesReadCount, err := precomputeHashes(ctx, layers, opts.workersCount)
	if err != nil {
		return v1.RootFS{}, bytesReadCount, fmt.Errorf("error occurrent while precomputing hashes: %w", err)
	}

	diffIDs, err := prepareDiffIDs(layers)
	if err != nil {
		return v1.RootFS{}, bytesReadCount, fmt.Errorf("failed to prepare diff ids: %w", err)
	}
	return v1.RootFS{
		Type:    "layers",
		DiffIDs: diffIDs,
	}, bytesReadCount, nil
}

func Read(ctx context.Context, dir string, opt ...Option) (*DirImage, error) {
	opts := makeOptions(opt...)
	cfgFile, err := prepareConfigFile(dir, opts.omitLayersContent)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare config file: %w", err)
	}

	layers, err := prepareLayers(dir, cfgFile, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare layers: %w", err)
	}
	var bytesReadCount int64
	cfgFile.RootFS, bytesReadCount, err = computeRootFS(ctx, layers, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to compute root filesystem: %w", err)
	}

	addendums, err := prepareAddendums(layers)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare addendums: %w", err)
	}
	img, err := prepareImage(cfgFile, addendums)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare image: %w", err)
	}
	return &DirImage{
		Image:          img,
		BytesReadCount: bytesReadCount,
		directory:      dir,
		// TODO: Descriptors
	}, nil
}
