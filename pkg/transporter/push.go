package transporter

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/tomekjarosik/geranos/pkg/image"
)

func Push(imageRef string, opt ...Option) error {

	opts := makeOptions(opt...)

	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return err
	}

	authenticator, err := authn.DefaultKeychain.Resolve(ref.Context().Registry)
	if err != nil {
		return err
	}
	lm := image.NewLayoutMapper(opts.imagesPath)

	img, err := lm.Read(ref)
	if err != nil {
		return fmt.Errorf("unable to read image from disk: %w", err)
	}
	if opts.mountedReference != nil {
		img = image.NewMountableImage(img, opts.mountedReference)
	}
	if err := remote.Write(ref, img, remote.WithAuth(authenticator)); err != nil {
		return fmt.Errorf("unable to push image to registry: %w", err)
	}
	return nil
}
