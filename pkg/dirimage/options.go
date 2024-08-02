package dirimage

import (
	"log"
	"runtime"
)

type options struct {
	workersCount             int
	chunkSize                int64
	printf                   func(fmt string, argv ...any)
	networkFailureRetryCount int
	progress                 chan<- ProgressUpdate
}

type Option func(opts *options)

func makeOptions(opts ...Option) *options {
	res := &options{
		workersCount:             min(8, runtime.NumCPU()),
		chunkSize:                64 * 1024 * 1024,
		printf:                   log.Printf,
		networkFailureRetryCount: 3,
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

func WithWorkersCount(workersCount int) Option {
	return func(o *options) {
		o.workersCount = workersCount
	}
}

func WithLogFunction(log func(fmt string, args ...any)) Option {
	return func(o *options) {
		o.printf = log
	}
}

func WithProgressChannel(progress chan<- ProgressUpdate) Option {
	return func(o *options) {
		o.progress = progress
	}
}
