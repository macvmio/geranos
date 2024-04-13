package sparsefile

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func TestWriteAndClose(t *testing.T) {
	t.Run("write only zeroes should defer write", func(t *testing.T) {
		tmpfile, err := ioutil.TempFile("", "sparsefile.test.*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name()) // clean up

		w := NewWriter(tmpfile)

		written, err := w.Write([]byte{0, 0, 0, 0})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if written != 4 {
			t.Errorf("expected to write 4 bytes, wrote %d", written)
		}

		if w.deferred != 4 {
			t.Errorf("expected deferred count to be 4, got %d", w.deferred)
		}

		if err := w.Close(); err != nil {
			t.Errorf("expected no error on close, got %v", err)
		}
	})

	t.Run("write with non-zeroes should perform write", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "sparsefile.test.*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name()) // clean up

		w := NewWriter(tmpfile)

		w.Write([]byte{0, 0, 0, 0}) // these are deferred
		written, err := w.Write([]byte{1, 2, 3, 4})
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if written != 4 {
			t.Errorf("expected to write 4 bytes, wrote %d", written)
		}

		if w.deferred != 0 {
			t.Errorf("expected deferred count to be 0, got %d", w.deferred)
		}

		if err := w.Close(); err != nil {
			t.Errorf("expected no error on close, got %v", err)
		}

		content, _ := os.ReadFile(tmpfile.Name())
		expected := append([]byte{0, 0, 0, 0}, []byte{1, 2, 3, 4}...)
		if !bytes.Equal(content, expected) {
			t.Errorf("file content mismatch, got %v, want %v", content, expected)
		}
	})
}
