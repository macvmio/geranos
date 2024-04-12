package dirimage

import (
	"log"
	"runtime"
)

type options struct {
	workersCount int
	chunkSize    int64
	printf       func(fmt string, argv ...any)
}

type Option func(opts *options)

func makeOptions(opts ...Option) *options {
	res := &options{
		workersCount: max(8, runtime.NumCPU()),
		chunkSize:    128 * 1024 * 1024,
		printf:       log.Printf,
	}

	for _, o := range opts {
		o(res)
	}

	return res
}

func WithChunkSize(chunkSize int64) Option {
	return func(o *options) {
		o.chunkSize = chunkSize
	}
}
