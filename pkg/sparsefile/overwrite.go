package sparsefile

import (
	"bytes"
	"fmt"
	"io"
)

const maxBufSize = 64 * 1024

func Overwrite(dst io.ReadWriteSeeker, src io.Reader) (written int64, skipped int64, err error) {
	srcBuf := make([]byte, maxBufSize)
	dstBuf := make([]byte, maxBufSize)
	return overwriteBuffer(dst, src, srcBuf, dstBuf)
}

func overwriteBuffer(dst io.ReadWriteSeeker, src io.Reader, srcBuf, dstBuf []byte) (written int64, skipped int64, err error) {
	var shiftedSrc []byte
	dstPos, err := dst.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to seek current: %w", err)
	}
	for {
		nrSrc, er1 := src.Read(srcBuf)
		if nrSrc == 0 && er1 == io.EOF {
			break
		}
		nrDst, _ := dst.Read(dstBuf[:nrSrc])
		nrMin := min(nrSrc, nrDst)
		if bytes.Equal(dstBuf[:nrMin], srcBuf[:nrMin]) {
			dstPos += int64(nrMin)
			skipped += int64(nrMin)
			shiftedSrc = srcBuf[nrMin:nrSrc]
		} else {
			shiftedSrc = srcBuf[0:nrSrc]
		}
		// rewind dstPost
		var er3 error
		dstPos, er3 = dst.Seek(dstPos, io.SeekStart)
		if er3 != nil {
			err = er3
			break
		}
		nw, ew := dst.Write(shiftedSrc)
		if ew != nil {
			err = ew
			break
		}
		dstPos += int64(nw)
		written += int64(nw)
		if er1 != nil {
			if er1 != io.EOF {
				err = er1
			}
			break
		}
	}
	return
}
