package dirimage

import (
	"crypto/rand"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/macvmio/geranos/pkg/filesegment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// GenerateRandomFile creates a file of the specified size filled with random bytes
// using io.CopyN for efficient copying.
func generateRandomFile(fileName string, size int64) error {
	// Open a file for writing, creating it with 0666 permissions if it does not exist
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	// Copy the specified amount of random data to the file
	// rand.Reader is a global, shared instance of a cryptographically secure random number generator
	if _, err := io.CopyN(file, rand.Reader, size); err != nil {
		return fmt.Errorf("error copying random data to file: %w", err)
	}

	return nil
}

func TestConvert_InvalidImageFormat(t *testing.T) {
	l, err := random.Layer(10, filesegment.MediaType)
	require.NoError(t, err)
	img := empty.Image
	img, err = mutate.Append(img, mutate.Addendum{
		Layer:       l,
		History:     v1.History{},
		Annotations: nil,
		MediaType:   filesegment.MediaType,
	})
	require.NoError(t, err)

	outImg, err := Convert(img)
	require.Error(t, err, "missing filename annotation")
	require.Nil(t, outImg)
}

func TestConvert_ValidImage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "content-matches-*")
	require.NoErrorf(t, err, "unable to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)

	err = generateRandomFile(filepath.Join(tempDir, "file1.img"), 100)
	require.NoError(t, err)

	img := empty.Image
	for i := 0; i < 10; i++ {
		layer, err := filesegment.NewLayer(filepath.Join(tempDir, "file1.img"), filesegment.WithRange(int64(i*10), int64(i*10+9)))
		require.NoError(t, err)

		img, err = mutate.Append(img, mutate.Addendum{
			Layer:       layer,
			History:     v1.History{},
			Annotations: layer.Annotations(),
			MediaType:   layer.GetMediaType(),
		})
		require.NoError(t, err)
	}

	outImg, err := Convert(img)
	assert.NotNil(t, outImg)
}
