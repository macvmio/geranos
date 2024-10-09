package filesegment

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func NewWriter(dir string, d *Descriptor) (*os.File, error) {
	f, err := os.OpenFile(filepath.Join(dir, d.filename), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("unable to open file '%v': %w", filepath.Join(dir, d.filename), err)
	}

	_, err = f.Seek(d.start, io.SeekStart)
	if err != nil {
		return nil, fmt.Errorf("error while seeking to position '%d': %w", d.start, err)
	}
	return f, nil
}
