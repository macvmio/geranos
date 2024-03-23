package transporter

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tomekjarosik/geranos/pkg/image"
)

func Adopt(src string, dst string, opt ...Option) error {
	opts := makeOptions(opt...)
	dstRef, err := name.ParseReference(dst, name.StrictValidation)
	if err != nil {
		return fmt.Errorf("unable to parse reference: %w", err)
	}
	lm := image.NewLayoutMapper(opts.imagesPath)
	return lm.Adopt(src, dstRef)
}
