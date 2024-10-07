package sparsefile

import (
	"io"
	"os"
)

type Writer struct {
	f        *os.File
	deferred int64
}

func (w *Writer) Write(p []byte) (n int, err error) {
	if isAllZeroes(p) {
		w.deferred += int64(len(p))
		return len(p), nil
	}
	if w.deferred > 0 {
		_, err := w.f.Seek(w.deferred, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		w.deferred = 0
	}
	return w.f.Write(p)
}

func (w *Writer) Close() error {
	toWriteCount := 0
	buf := make([]byte, 1)
	if w.deferred > 0 {
		toWriteCount = 1
		w.deferred -= 1
		_, err := w.f.Seek(w.deferred, io.SeekCurrent)
		if err != nil {
			return err
		}
	}
	if toWriteCount > 0 {
		n, err := w.f.Write(buf[:toWriteCount])
		if err != nil {
			return err
		}
		if n != toWriteCount {
			return io.ErrShortWrite
		}
	}
	return w.f.Close()
}

func NewWriter(f *os.File) *Writer {
	return &Writer{
		f:        f,
		deferred: 0,
	}
}
