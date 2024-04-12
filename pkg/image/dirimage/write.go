package dirimage

import (
	"context"
	"errors"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/tomekjarosik/geranos/pkg/image/filesegment"
	"github.com/tomekjarosik/geranos/pkg/image/sparsefile"
	"golang.org/x/sync/errgroup"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
)

const LocalManifestFilename = ".oci.manifest.json"

func contentMatches(destinationDir string, segment *filesegment.Descriptor) bool {
	fname := filepath.Join(destinationDir, segment.Filename())
	f, err := os.OpenFile(fname, os.O_RDONLY, 0666)
	if err != nil {
		return false
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			// TODO: lm.opts.printf("error while closing file %v, got %v", segment.Filename(), err)
		}
	}(f)
	l, err := filesegment.NewLayer(fname, filesegment.WithRange(segment.Start(), segment.Stop()))
	if err != nil {
		return false
	}
	d, err := l.Digest()
	if err != nil {
		return false
	}
	if d == segment.Digest() {
		return true
	}
	return false
}

func writeToSegment(destinationDir string, segment *filesegment.Descriptor, src io.ReadCloser) (written int64, skipped int64, err error) {
	// Here: we have io.ReadCloser dumping to a file at given location
	f, err := filesegment.NewWriter(destinationDir, segment)
	if err != nil {
		return 0, 0, err
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			// TODO: opts.printf("error while closing file %v, got %v", segment.Filename(), err)
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

func (di *DirImage) Write(ctx context.Context, destinationDir string, opt ...Option) error {
	opts := makeOptions(opt...)

	type Job struct {
		Descriptor filesegment.Descriptor
		Layer      v1.Layer
	}
	type JobResult struct {
		Job Job
		err error
	}

	workersCount := min(opts.workersCount, runtime.NumCPU())
	jobs := make(chan Job, workersCount)
	results := make(chan JobResult, workersCount)

	g, ctx := errgroup.WithContext(ctx)

	for w := 0; w < workersCount; w++ {
		g.Go(func() error {
			for job := range jobs {
				atomic.AddInt64(&di.BytesReadCount, job.Descriptor.Length())
				if contentMatches(destinationDir, &job.Descriptor) {
					continue
				}
				var jobErr error
				for i := 0; i < opts.networkFailureRetryCount; i++ {
					written, skipped, err := writeLayer(destinationDir, &job.Descriptor, job.Layer)
					opts.printf("written=%d, skipped=%d\n", written, skipped)

					atomic.AddInt64(&di.BytesWrittenCount, written)
					atomic.AddInt64(&di.BytesSkippedCount, skipped)
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
		for _, d := range di.segmentDescriptors {
			select {
			case <-ctx.Done():
				return ctx.Err() // Early return on context cancellation.
			default:
				l, err := di.Image.LayerByDigest(d.Digest())
				if err != nil {
					opts.printf("invalid seg.Digest")
					l = nil
				}
				jobs <- Job{Descriptor: *d, Layer: l}
			}
		}
		return nil
	})
	go func() {
		err := g.Wait()
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
	rawManifest, err := di.Image.RawManifest()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(destinationDir, LocalManifestFilename), rawManifest, 0o777)
}
