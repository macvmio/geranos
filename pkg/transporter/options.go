package transporter

import (
	"context"
	"github.com/google/go-containerregistry/pkg/authn"
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
	workersCount     int
	ctx              context.Context
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

func WithWorkersCount(workersCount int) Option {
	return func(o *options) {
		o.workersCount = workersCount
	}
}

func WithContext(ctx context.Context) Option {
	return func(o *options) {
		o.ctx = ctx
		o.remoteOptions = append(o.remoteOptions, remote.WithContext(ctx))
	}
}

func makeOptions(opts ...Option) *options {
	res := options{
		imagesPath:       mustExpandUser("~/.geranos/images"),
		cachePath:        mustExpandUser("~/.geranos/cache"),
		mountedReference: nil,
		insecure:         false,
		remoteOptions: []remote.Option{
			remote.WithAuthFromKeychain(authn.DefaultKeychain),
		},
		refValidation: name.StrictValidation,
		workersCount:  8,
		ctx:           context.Background(),
	}
	for _, o := range opts {
		o(&res)
	}
	return &res
}
