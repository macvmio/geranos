package transporter

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/macvmio/geranos/pkg/layout"
	"golang.org/x/sync/errgroup"
	"log"
	"os"
)

func prePushConcurrently(repo name.Repository, img v1.Image, opts *options) error {
	layers, err := img.Layers()
	if err != nil {
		return fmt.Errorf("unable to extract layers from image: %w", err)
	}
	g, _ := errgroup.WithContext(opts.ctx)
	g.SetLimit(opts.workersCount)

	seen := make(map[string]bool, 0)
	for _, l := range layers {
		currentLayer := l
		h, err := currentLayer.Digest()
		if err != nil {
			return err
		}
		if _, ok := seen[h.String()]; ok {
			continue
		}
		seen[h.String()] = true
		g.Go(func() error {
			log.Printf("pushing layer: %v", h)
			return remote.WriteLayer(repo, currentLayer, opts.remoteOptions...)
		})
	}
	err = g.Wait()
	if err != nil {
		return fmt.Errorf("error occured while pushing layers concurrently: %w", err)
	}
	return nil
}

func Push(imageRef string, opt ...Option) error {
	logs.Progress = log.New(os.Stdout, "", log.LstdFlags)
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

	if opts.workersCount > 0 {
		err := prePushConcurrently(ref.Context(), img, opts)
		if err != nil {
			return err
		}
	}

	if err := remote.Write(ref, img, opts.remoteOptions...); err != nil {
		return fmt.Errorf("unable to push image to registry: %w", err)
	}
	return nil
}
