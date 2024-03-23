package transporter

import (
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type options struct {
	imagesPath       string
	cachePath        string
	mountedReference name.Reference
	insecure         bool
	remoteOptions    []remote.Option
	refValidation    name.Option
}

type Option func(opts *options)

func WithImagesPath(imagesPath string) Option {
	return func(o *options) {
		o.imagesPath = imagesPath
	}
}

func WithInsecureTransport() Option {
	return func(o *options) {
		o.insecure = false
	}
}

func WithMountedReference(ref name.Reference) Option {
	return func(o *options) {
		o.mountedReference = ref
	}
}

func makeOptions(opts ...Option) *options {
	res := options{
		imagesPath:       mustExpandUser("~/.geranos/images"),
		cachePath:        "",
		mountedReference: nil,
		insecure:         false,
		remoteOptions:    nil,
		refValidation:    name.StrictValidation,
	}
	for _, o := range opts {
		o(&res)
	}
	return &res
}
