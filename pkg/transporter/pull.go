package transporter

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/mobileinf/geranos/pkg/dirimage"
	"github.com/mobileinf/geranos/pkg/layout"
	"log"
	"strings"
)

func updateProgress(progress int64) {
	buffer := new(strings.Builder)
	fmt.Fprintf(buffer, "\rProgress: [%-100s] %d%%", strings.Repeat("=", int(progress)), progress)
	fmt.Print(buffer.String())
}

func printProgress(progress <-chan dirimage.ProgressUpdate) {
	last := int64(0)
	for p := range progress {
		current := 100 * p.BytesProcessed / p.BytesTotal
		if current != last {
			updateProgress(current)
		}
		last = current
	}
	fmt.Printf("\n")
}

func Pull(src string, opt ...Option) error {
	opts := makeOptions(opt...)
	ref, err := name.ParseReference(src, name.StrictValidation)
	if err != nil {
		return err
	}
	img, err := remote.Image(ref, opts.remoteOptions...)
	if err != nil {
		return err
	}
	// Cache is not important if Sketch is working properly
	//img = cache.Image(img, diskcache.NewFilesystemCache(opts.cachePath))
	progress := make(chan dirimage.ProgressUpdate)
	defer close(progress)

	dirimageOptions := []dirimage.Option{
		dirimage.WithProgressChannel(progress),
		dirimage.WithLogFunction(func(fmt string, args ...any) {
		}),
	}
	if opts.verbose {
		dirimageOptions = append(dirimageOptions, dirimage.WithLogFunction(log.Printf))
	}

	lm := layout.NewMapper(opts.imagesPath, dirimageOptions...)
	go printProgress(progress)
	return lm.Write(opts.ctx, img, ref)
}
