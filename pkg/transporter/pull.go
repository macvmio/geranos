package transporter

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tomekjarosik/geranos/pkg/image"
)

func Pull(src string, opt ...Option) error {
	opts := makeOptions(opt...)
	ref, err := name.ParseReference(src, name.StrictValidation)
	if err != nil {
		return err
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return err
	}
	lm := image.NewLayoutMapper(opts.imagesPath)
	return lm.Write(img, ref, make(chan image.ProgressUpdate))
}
