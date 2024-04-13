package filesegment

import (
	"fmt"
	"os"
)

func Split(fullpath string, chunkSize int64, opt ...LayerOpt) ([]*Layer, error) {
	f, err := os.Stat(fullpath)
	if err != nil {
		return nil, fmt.Errorf("faild to stat file '%v': %w", fullpath, err)
	}
	if f.Size() < chunkSize {
		l, err := NewLayer(fullpath, opt...)
		if err != nil {
			return nil, err
		}
		return []*Layer{l}, nil
	}
	res := make([]*Layer, 0)
	maxIdx := f.Size() - 1

	for start := int64(0); start <= maxIdx; start += chunkSize {
		stop := start + chunkSize - 1
		if stop > maxIdx {
			stop = maxIdx
		}
		l, err := NewLayer(fullpath, append(opt, WithRange(start, stop))...)
		if err != nil {
			return nil, err
		}
		res = append(res, l)
	}

	return res, nil
}
