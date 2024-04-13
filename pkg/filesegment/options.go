package filesegment

type LayerOpt func(*Layer)

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
