package image

type options struct {
	workersCount int
}

type Option func(opts *options)

func WithWorkersCount(count int) Option {
	return func(opts *options) {
		opts.workersCount = count
	}
}
