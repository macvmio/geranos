package zstd

import (
	"bytes"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

// Test_ReadCloser verifies that ReadCloser compresses the input correctly.
func Test_ReadCloser(t *testing.T) {
	// Create a buffer filled with 327680 bytes of zeroes as input
	inputData := bytes.Repeat([]byte{0}, 327680)
	input := io.NopCloser(bytes.NewReader(inputData))

	// Use ReadCloser to compress the data
	compressedReader := ReadCloser(input)
	defer compressedReader.Close()

	h, n, err := v1.SHA256(compressedReader)
	require.NoError(t, err)
	assert.Equal(t, v1.Hash{
		Algorithm: "sha256",
		Hex:       "ffc1459590bc9dee536a1fb0a6fee5b31beefa35348488541c343c1d3af41f5d",
	}, h)
	assert.Equal(t, int64(78), n)
}
