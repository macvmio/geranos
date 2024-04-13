package filesegment

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// Helper function to create a temp file with specific size
func createTempFile(t *testing.T, dir string, size int64) string {
	t.Helper()
	file, err := os.CreateTemp(dir, "prefix-")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer file.Close()

	if size > 0 {
		data := make([]byte, size)
		if _, err := file.Write(data); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}

	return file.Name()
}

func TestSplit(t *testing.T) {
	dir, err := os.MkdirTemp("", "split-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(dir) // Clean up after the test

	// Test with empty file
	t.Run("Empty file", func(t *testing.T) {
		filename := createTempFile(t, dir, 0)
		layers, err := Split(filename, 1024)
		assert.Error(t, err)
		assert.Nil(t, layers)
	})

	// Test with file size smaller than chunk size
	t.Run("File smaller than chunk size", func(t *testing.T) {
		filename := createTempFile(t, dir, 500) // Smaller than 1024 bytes
		layers, err := Split(filename, 1024)
		assert.NoError(t, err)
		assert.Len(t, layers, 1, "Should have one layer")
		assert.Equal(t, int64(0), layers[0].start)
		assert.Equal(t, int64(499), layers[0].stop)
	})

	// Test with file size exactly equal to chunk size
	t.Run("File size equals chunk size", func(t *testing.T) {
		filename := createTempFile(t, dir, 1024)
		layers, err := Split(filename, 1024)
		assert.NoError(t, err)
		assert.Len(t, layers, 1, "Should split into exactly one layer")
		assert.Equal(t, int64(0), layers[0].start)
		assert.Equal(t, int64(1023), layers[0].stop)
	})

	// Test with file size greater than chunk size
	t.Run("Multiple chunks", func(t *testing.T) {
		filename := createTempFile(t, dir, 2050) // A bit more than two chunks of 1024 bytes
		layers, err := Split(filename, 1024)
		assert.NoError(t, err)
		assert.Len(t, layers, 3, "Should split into three layers")
		assert.Equal(t, int64(0), layers[0].start)
		assert.Equal(t, int64(1023), layers[0].stop)
		assert.Equal(t, int64(1024), layers[1].start)
		assert.Equal(t, int64(2047), layers[1].stop)
		assert.Equal(t, int64(2048), layers[2].start)
		assert.Equal(t, int64(2049), layers[2].stop)
	})
}
