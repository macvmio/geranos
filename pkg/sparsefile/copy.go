package sparsefile

import (
	"io"
)

const maxBufSize = 64 * 1024

func Copy(dst io.WriteSeeker, src io.Reader) (written int64, skipped int64, err error) {
	return copySparseBuffer(dst, src, nil)
}

func copySparseBuffer(dst io.WriteSeeker, src io.Reader, buf []byte) (written int64, skipped int64, err error) {
	if buf == nil {
		size := maxBufSize
		buf = make([]byte, size)
	}
	var deferred int64
	for {
		nr, er := src.Read(buf)
		if er == nil && isAllZeroes(buf[:nr]) {
			deferred += int64(nr)
			continue
		}
		if deferred > 0 {
			if nr == 0 {
				deferred -= 1
				nr = 1
				buf[0] = 0
			}
			_, ers := dst.Seek(deferred, io.SeekCurrent)
			skipped += deferred
			deferred = 0
			if ers != nil {
				err = ers
			}
		}
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if ew != nil {
				err = ew
				break
			}
			written += int64(nw)
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, skipped, err
}
