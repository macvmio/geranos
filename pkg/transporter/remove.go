package transporter

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tomekjarosik/geranos/pkg/image"
)

func Remove(src string, opt ...Option) error {
	opts := makeOptions(opt...)
	ref, err := name.ParseReference(src, name.StrictValidation)
	if err != nil {
		return fmt.Errorf("unable to parse reference: %w", err)
	}
	lm := image.NewLayoutMapper(opts.imagesPath)
	return lm.Remove(ref)
}
