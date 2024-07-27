package transporter

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tomekjarosik/geranos/pkg/layout"
	"golang.org/x/sync/errgroup"
)

func Push(imageRef string, opt ...Option) error {

	opts := makeOptions(opt...)

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return fmt.Errorf("unable to parse reference '%v': %w", imageRef, err)
	}

	lm := layout.NewMapper(opts.imagesPath)

	img, err := lm.Read(opts.ctx, ref)
	if err != nil {
		return fmt.Errorf("unable to read image from disk: %w", err)
	}
	if opts.mountedReference != nil {
		img = layout.NewMountableImage(img, opts.mountedReference)
	}

	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("unable to extract layers from image: %w", err)
	}
	g, _ := errgroup.WithContext(opts.ctx)
	g.SetLimit(opts.workersCount)
	for _, l := range layers {
		currentLayer := l
		g.Go(func() error {
			fmt.Printf("pushing %v\n", currentLayer)
			return remote.WriteLayer(ref.Context(), currentLayer, opts.remoteOptions...)
		})
	}
	err = g.Wait()
	if err != nil {
		return fmt.Errorf("error occured while pushing layers concurrently: %w", err)
	}

	if err := remote.Write(ref, img, opts.remoteOptions...); err != nil {
		return fmt.Errorf("unable to push image to registry: %w", err)
	}
	return nil
}
