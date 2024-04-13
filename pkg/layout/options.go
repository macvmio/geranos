package layout

import (
	"log"
	"runtime"
)

type options struct {
	workersCount             int
	chunkSize                int64
	printf                   func(fmt string, argv ...any)
	networkFailureRetryCount int
}

type Option func(opts *options)

func makeOptions(opts ...Option) *options {
	res := &options{
		workersCount:             min(8, runtime.NumCPU()),
		chunkSize:                128 * 1024 * 1024,
		printf:                   log.Printf,
		networkFailureRetryCount: 3,
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

func WithLogFunction(log func(fmt string, argv ...any)) Option {
	return func(opts *options) {
		opts.printf = log
	}
}
