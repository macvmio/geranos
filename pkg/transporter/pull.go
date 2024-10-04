package transporter

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/macvmio/geranos/pkg/layout"
)

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
	lm := layout.NewMapper(opts.imagesPath, opts.dirimageOptions...)
	if opts.force {
		return lm.Write(opts.ctx, img, ref)
	}
	return lm.WriteIfNotPresent(opts.ctx, img, ref)
}
