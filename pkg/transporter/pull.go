package transporter

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tomekjarosik/geranos/pkg/diskcache"
	"github.com/tomekjarosik/geranos/pkg/layout"
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
	img = cache.Image(img, diskcache.NewFilesystemCache(opts.cachePath))
	lm := layout.NewMapper(opts.imagesPath)
	return lm.Write(opts.ctx, img, ref)
}
