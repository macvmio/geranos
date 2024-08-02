package dirimage

import (
	"context"
	"errors"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/tomekjarosik/geranos/pkg/filesegment"
	"github.com/tomekjarosik/geranos/pkg/sparsefile"
	"golang.org/x/sync/errgroup"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"syscall"
)

const LocalManifestFilename = ".oci.manifest.json"

func writeToSegment(destinationDir string, segment *filesegment.Descriptor, src io.ReadCloser) (written int64, skipped int64, err error) {
	// Here: we have io.ReadCloser dumping to a file at given location
	f, err := filesegment.NewWriter(destinationDir, segment)
	if err != nil {
		return 0, 0, err
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("error while closing file %v, got %v", segment.Filename(), err)
		}
	}(f)

	written, skipped, err = sparsefile.Copy(f, src)
	if written+skipped != segment.Length() {
		return written, skipped, fmt.Errorf("invalid numer of bytes written+skipped: segment length: %d, written+skipped: %d", segment.Length(), written+skipped)
	}
	return written, skipped, err
}

func writeLayer(destinationDir string, segment *filesegment.Descriptor, layer v1.Layer) (written int64, skipped int64, err error) {
	if layer == nil {
		return 0, 0, errors.New("nil layer provided")
	}

	rc, err := layer.Uncompressed()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to access uncompressed layer: %w", err)
	}
	defer rc.Close()
	return writeToSegment(destinationDir, segment, rc)
}

func truncateFiles(destinationDir string, segmentDescriptors []*filesegment.Descriptor) error {
	fileSizesMap := make(map[string]int64)
	for _, d := range segmentDescriptors {
		size, present := fileSizesMap[d.Filename()]
		if !present {
			size = d.Stop() + 1
		}
		size = max(size, d.Stop()+1)
		fileSizesMap[d.Filename()] = size
	}

	for filename, size := range fileSizesMap {
		err := os.Truncate(filepath.Join(destinationDir, filename), size)
		if err != nil {
			return fmt.Errorf("error while truncating file '%v': %w", filename, err)
		}
	}
	return nil
}

func sendProgressUpdate(progressChan chan<- ProgressUpdate, current, total int64) {
	select {
	case progressChan <- ProgressUpdate{
		BytesProcessed: current,
		BytesTotal:     total,
	}:
	default:
	}
}

func (di *DirImage) Write(ctx context.Context, destinationDir string, opt ...Option) error {
	if di.Image == nil {
		return errors.New("invalid image")
	}
	opts := makeOptions(opt...)

	type Job struct {
		Descriptor filesegment.Descriptor
		Layer      v1.Layer
	}
	type JobResult struct {
		Job Job
		err error
	}
	bytesTotal := di.Length()
	jobs := make(chan Job, opts.workersCount)
	g, ctx := errgroup.WithContext(ctx)
	layerOpts := []filesegment.LayerOpt{filesegment.WithLogFunction(opts.printf)}
	for w := 0; w < opts.workersCount; w++ {
		g.Go(func() error {
			for job := range jobs {
				atomic.AddInt64(&di.BytesReadCount, job.Descriptor.Length())
				sendProgressUpdate(opts.progress, di.BytesReadCount, bytesTotal)
				if filesegment.Matches(&job.Descriptor, destinationDir, layerOpts...) {
					opts.printf("existing layer: %v\n", &job.Descriptor)
					continue
				}

				for i := 0; i < opts.networkFailureRetryCount; i++ {
					written, skipped, err := writeLayer(destinationDir, &job.Descriptor, job.Layer)
					opts.printf("downloaded layer: %v, written=%d, skipped=%d\n", &job.Descriptor, written, skipped)

					atomic.AddInt64(&di.BytesWrittenCount, written)
					atomic.AddInt64(&di.BytesSkippedCount, skipped)
					if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.EPIPE) {
						continue
					}
					if err == nil {
						break
					}
					opts.printf("failed writing to file '%v' at offset '%v': %v\n", job.Descriptor.Filename(), job.Descriptor.Start(), err)
				}
			}
			return nil
		})
	}

	g.Go(func() error {
		defer close(jobs)
		for _, d := range di.segmentDescriptors {
			l, err := di.Image.LayerByDigest(d.Digest())
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err() // Early return on context cancellation.
			case jobs <- Job{Descriptor: *d, Layer: l}:
			}
		}
		return nil
	})

	err := g.Wait()
	if err != nil {
		return err
	}

	rawManifest, err := di.Image.RawManifest()
	if err != nil {
		return err
	}

	err = truncateFiles(destinationDir, di.segmentDescriptors)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(destinationDir, LocalManifestFilename), rawManifest, 0o777)
}
