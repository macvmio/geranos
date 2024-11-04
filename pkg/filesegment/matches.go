package filesegment

import (
	"path/filepath"
)

func Matches(d *Descriptor, dir string, opt ...LayerOpt) bool {
	fname := filepath.Join(dir, d.filename)
	l, err := NewLayer(fname, append(opt, WithRange(d.start, d.stop))...)
	if err != nil {
		return false
	}
	diffID, err := l.DiffID()
	return err == nil && diffID == d.diffID
}
