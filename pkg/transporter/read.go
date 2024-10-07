package transporter

import (
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/macvmio/geranos/pkg/layout"
)

func Read(src string, opt ...Option) (v1.Image, error) {
	opts := makeOptions(opt...)
	ref, err := name.ParseReference(src, name.StrictValidation)
	if err != nil {
		return nil, err
	}
	lm := layout.NewMapper(opts.imagesPath, opts.dirimageOptions...)
	return lm.Read(opts.ctx, ref)
}
