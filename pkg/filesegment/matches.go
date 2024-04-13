package filesegment

import "path/filepath"

func Matches(d *Descriptor, dir string, opt ...LayerOpt) bool {
	fname := filepath.Join(dir, d.filename)
	l, err := NewLayer(fname, append(opt, WithRange(d.start, d.stop))...)
	if err != nil {
		return false
	}
	digest, err := l.Digest()
	if err != nil {
		return false
	}
	if digest == d.digest {
		return true
	}
	return false
}
