package filesegment

import "github.com/google/go-containerregistry/pkg/v1/types"

type LayerOpt func(*Layer)

func WithMediaType(mt types.MediaType) LayerOpt {
	return func(l *Layer) {
		l.mediaType = mt
	}
}

func WithRange(start, stop int64) LayerOpt {
	return func(l *Layer) {
		l.start = start
		l.stop = stop
	}
}

func WithLogFunction(log func(fmt string, args ...any)) LayerOpt {
	return func(l *Layer) {
		l.log = log
	}
}
