package segmentlayer

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type partialFileReader struct {
	filename string
	start    int64
	stop     int64 // inclusive end of interval
	offset   int64

	r *bufio.Reader
	f *os.File
}

func (pfr *partialFileReader) open() error {
	f, err := os.Open(pfr.filename)
	if err != nil {
		return err
	}
	info, err := os.Stat(pfr.filename)
	if err != nil {
		return fmt.Errorf("unable to state a file '%v': %w", pfr.filename, err)
	}
	if pfr.stop >= info.Size() {
		pfr.stop = info.Size() - 1
	}
	pfr.offset = pfr.start
	_, err = f.Seek(pfr.start, io.SeekStart)
	if err != nil {
		return err
	}
	pfr.f = f
	pfr.r = bufio.NewReaderSize(f, 64*1024)

	return nil
}

func (pfr *partialFileReader) Close() error {
	err := pfr.f.Close()
	pfr.f = nil
	pfr.r = nil
	return err
}
func (pfr *partialFileReader) Read(p []byte) (n int, err error) {
	if pfr.r == nil {
		err = pfr.open()
		if err != nil {
			return 0, err
		}
	}
	n, err = pfr.r.Read(p)
	if err == io.EOF {
		if pfr.offset+int64(n) > pfr.stop {
			n = int(pfr.stop - pfr.offset + 1)
		}
		pfr.offset += int64(n)
		return n, io.EOF
	}
	if pfr.offset+int64(n) > pfr.stop {
		n = int(pfr.stop - pfr.offset + 1)
		err = io.EOF
	}
	pfr.offset += int64(n)
	return n, err
}
