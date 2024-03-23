package transporter

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tomekjarosik/geranos/pkg/image"
)

func Clone(src string, dst string, opt ...Option) error {
	opts := makeOptions(opt...)
	srcRef, err := name.ParseReference(src, name.StrictValidation)
	if err != nil {
		return err
	}
	dstRef, err := name.ParseReference(dst, name.StrictValidation)
	if err != nil {
		return err
	}

	lm := image.NewLayoutMapper(opts.imagesPath)
	return lm.Clone(srcRef, dstRef)
}
