package transporter

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/macvmio/geranos/pkg/layout"
)

func Rehash(src string, opt ...Option) error {
	opts := makeOptions(opt...)
	ref, err := name.ParseReference(src, name.StrictValidation)
	if err != nil {
		return fmt.Errorf("parse ref %s: %v", src, err)
	}
	lm := layout.NewMapper(opts.imagesPath, opts.dirimageOptions...)
	return lm.Rehash(opts.ctx, ref)
}
