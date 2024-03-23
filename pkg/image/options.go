package image

type options struct {
	workersCount int
	chunkSize    int64
}

type Option func(opts *options)

func makeOptions(opts ...Option) *options {
	res := &options{
		workersCount: 8,
		chunkSize:    128 * 1024 * 1024,
	}

	for _, o := range opts {
		o(res)
	}

	return res
}

func WithWorkersCount(count int) Option {
	return func(opts *options) {
		opts.workersCount = count
	}
}

func WithChunkSize(chunkSize int64) Option {
	return func(opts *options) {
		opts.chunkSize = chunkSize
	}
}
