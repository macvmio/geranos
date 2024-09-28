package transporter

import (
	"context"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/mobileinf/geranos/pkg/dirimage"
	"log"
)

type options struct {
	imagesPath       string
	cachePath        string
	mountedReference name.Reference
	insecure         bool
	remoteOptions    []remote.Option
	dirimageOptions  []dirimage.Option
	refValidation    name.Option
	workersCount     int
	verbose          bool
	force            bool
	ctx              context.Context
}

type Option func(opts *options)

func WithImagesPath(imagesPath string) Option {
	return func(o *options) {
		o.imagesPath = imagesPath
	}
}

func WithCachePath(cachePath string) Option {
	return func(o *options) {
		o.cachePath = cachePath
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

func WithVerbose(verbose bool) Option {
	return func(o *options) {
		o.verbose = verbose
		if verbose {
			o.dirimageOptions = append(o.dirimageOptions, dirimage.WithLogFunction(log.Printf))
		} else {
			o.dirimageOptions = append(o.dirimageOptions, dirimage.WithLogFunction(func(fmt string, args ...any) {
			}))
		}
	}
}

func WithForce(force bool) Option {
	return func(o *options) {
		o.force = force
	}
}

func WithProgressChannel(c chan<- ProgressUpdate) Option {
	return func(o *options) {
		// Create a new dirimage channel to be used internally
		dirimageChan := make(chan dirimage.ProgressUpdate)

		// Start a goroutine to convert dirimage.ProgressUpdate to ProgressUpdate
		go func() {
			for progress := range dirimageChan {
				// Convert ProgressUpdate to dirimage.ProgressUpdate and send it
				c <- ProgressUpdate{
					BytesProcessed: progress.BytesProcessed,
					BytesTotal:     progress.BytesTotal,
				}
			}
		}()
		o.dirimageOptions = append(o.dirimageOptions, dirimage.WithProgressChannel(dirimageChan))
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
		dirimageOptions: []dirimage.Option{},
		refValidation:   name.StrictValidation,
		workersCount:    8,
		verbose:         false,
		ctx:             context.Background(),
	}
	for _, o := range opts {
		o(&res)
	}
	return &res
}
