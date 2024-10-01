package dirimage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/mobileinf/geranos/pkg/filesegment"
	"golang.org/x/sync/errgroup"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

func precompute(ctx context.Context, layers []*filesegment.Layer, workersCount int) (bytesReadCount int64, err error) {
	jobs := make(chan *filesegment.Layer, workersCount)
	g, ctx := errgroup.WithContext(ctx)

	var aBytesReadCount atomic.Int64
	for w := 0; w < workersCount; w++ {
		g.Go(func() error {
			for l := range jobs {
				_, _ = l.DiffID()
				_, _ = l.Digest()
				aBytesReadCount.Add(2 * l.Length())
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

func Read(ctx context.Context, dir string, opt ...Option) (*DirImage, error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory: '%v': %w", dir, err)
	}
	opts := makeOptions(opt...)

	layers := make([]*filesegment.Layer, 0)
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
		layers = append(layers, fileLayers...)
	}

	bytesReadCount, err := precompute(ctx, layers, opts.workersCount)
	if err != nil {
		return nil, fmt.Errorf("error occurrent while precomputing hashes: %w", err)
	}
	addendums := make([]mutate.Addendum, 0)
	diffIDs := make([]v1.Hash, 0)
	for _, l := range layers {
		addendums = append(addendums, mutate.Addendum{
			Layer:       l,
			History:     v1.History{},
			Annotations: l.Annotations(),
			MediaType:   l.GetMediaType(),
		})
		diffID, err := l.DiffID()
		if err != nil {
			return nil, err
		}
		diffIDs = append(diffIDs, diffID)
	}
	img := empty.Image
	img, err = mutate.Append(img, addendums...)
	if err != nil {
		return nil, fmt.Errorf("unable to append layers to image: %w", err)
	}

	img, err = mutate.ConfigFile(img, &v1.ConfigFile{
		Container: "geranos",
		RootFS: v1.RootFS{
			Type:    "layers",
			DiffIDs: diffIDs,
		},
		Created: v1.Time{Time: time.Now()},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to mutate config file: %w", err)
	}
	img = mutate.MediaType(img, ManifestMediaType)
	img = mutate.ConfigMediaType(img, ConfigMediaType)
	return &DirImage{
		Image:          img,
		BytesReadCount: bytesReadCount,
		directory:      dir,
		// TODO: Descriptors
	}, nil
}

func ReadManifest(dir string) (*v1.Manifest, error) {
	file, err := os.Open(filepath.Join(dir, LocalManifestFilename))
	if err != nil {
		return nil, fmt.Errorf("unable to open manifest file: %w", err)
	}
	defer file.Close()
	return v1.ParseManifest(file)
}

// ReadDigest reads the digest from the manifest in the specified directory.
func ReadDigest(dir string) (v1.Hash, error) {
	// Use ReadManifest to get the manifest
	manifest, err := ReadManifest(dir)
	if err != nil {
		return v1.Hash{}, fmt.Errorf("unable to read manifest: %w", err)
	}

	// Serialize the manifest to JSON
	jsonData, err := json.Marshal(manifest)
	if err != nil {
		return v1.Hash{}, fmt.Errorf("failed to marshal manifest to JSON: %w", err)
	}
	hash, _, err := v1.SHA256(bytes.NewBuffer(jsonData))
	return hash, err
}
