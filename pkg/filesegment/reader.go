package filesegment

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

type partialFileReader struct {
	f *os.File
	r *bufio.Reader
}

func newPartialFileReader(filepath string, start, stop int64) (*partialFileReader, error) {
	size := stop - start + 1
	if size <= 0 {
		return nil, fmt.Errorf("invalid range: start (%d) must be less than or equal to stop (%d)", start, stop)
	}
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	fileInfo, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if start >= fileInfo.Size() {
		f.Close()
		return nil, fmt.Errorf("start position (%d) is beyond file size (%d)", start, fileInfo.Size())
	}
	if stop >= fileInfo.Size() {
		stop = fileInfo.Size() - 1
		size = stop - start + 1
	}
	sr := io.NewSectionReader(f, start, size)
	pfr := &partialFileReader{
		f: f,
		r: bufio.NewReaderSize(sr, 512*1024),
	}
	return pfr, nil
}

func (pfr *partialFileReader) Read(p []byte) (n int, err error) {
	return pfr.r.Read(p)
}

func (pfr *partialFileReader) Close() error {
	pfr.r = nil
	return pfr.f.Close()
}
