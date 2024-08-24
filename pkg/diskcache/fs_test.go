package diskcache

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/stream"
	"github.com/mobileinf/geranos/pkg/filesegment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func Test_FSCache_Put_Compressed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "diskcache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	c := NewFilesystemCache(tempDir)

	var n, wantSize int64 = 10000, 49
	newBlob := func() io.ReadCloser { return io.NopCloser(bytes.NewReader(bytes.Repeat([]byte{'a'}, int(n)))) }
	wantDigest := "sha256:3d7c465be28d9e1ed810c42aeb0e747b44441424f566722ba635dc93c947f30e"
	wantDiffId := "sha256:27dd1f61b867b6a0f6e9d8a41c43231de52107e53ae424de8f847b821db4b711"

	originalLayer := stream.NewLayer(newBlob())
	compressedLayer, err := originalLayer.Compressed()
	if _, err := io.Copy(io.Discard, compressedLayer); err != nil {
		t.Errorf("error reading compressed: %v", err)
	}

	l, err := c.Put(originalLayer)
	require.NoError(t, err)
	assert.NotNil(t, l)

	if d, err := l.Digest(); err != nil {
		t.Errorf("Digest: %v", err)
	} else if d.String() != wantDigest {
		t.Errorf("stream Digest got %q, want %q", d.String(), wantDigest)
	}
	if d, err := l.DiffID(); err != nil {
		t.Errorf("DiffID: %v", err)
	} else if d.String() != wantDiffId {
		t.Errorf("stream diffID got %q, want %q", d.String(), wantDiffId)
	}

	if s, err := l.Size(); err != nil {
		t.Errorf("Size: %v", err)
	} else if s != wantSize {
		t.Errorf("stream Size got %q, want %d", s, wantSize)
	}
}

func Test_FSCache_PutAndGet_Uncompressed_WithTarCompressed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "diskcache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	c := NewFilesystemCache(tempDir)

	img, err := random.Image(1024*100, 8)
	require.NoError(t, err)

	layers, err := img.Layers()
	require.NoError(t, err)

	for layerID := 0; layerID < 8; layerID++ {

		lCache, err := c.Put(layers[layerID])
		require.NoError(t, err)
		assert.NotNil(t, lCache)

		if uncomp, err := lCache.Uncompressed(); err != nil {
			t.Errorf("Compressed: %v", err)
		} else {
			if _, err := io.Copy(io.Discard, uncomp); err != nil {
				t.Errorf("error reading compressed: %v", err)
			}
			err = uncomp.Close()
			require.NoError(t, err)
		}
		diffID, err := layers[layerID].DiffID()
		require.NoError(t, err)
		lCache, err = c.Get(diffID)
		require.NoError(t, err)

		digest, err := layers[layerID].Digest()
		lCache, err = c.Get(digest)
		// random.Image() generates tar compression, which is not supported as fscache uses zstd
		// so the layer cannot be found
		require.ErrorIs(t, err, cache.ErrNotFound)
	}
}

// Compressed at source is not supported
/*
func Test_FSCache_PutAndGet_CompressedAtSource(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "diskcache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	c := NewFilesystemCache(tempDir)

	img, err := random.Image(1024*100, 8)
	require.NoError(t, err)

	layers, err := img.Layers()
	require.NoError(t, err)

	for layerID := 0; layerID < 8; layerID++ {

		lCache, err := c.Put(layers[layerID])
		require.NoError(t, err)
		assert.NotNil(t, lCache)

		if comp, err := lCache.Compressed(); err != nil {
			t.Errorf("Compressed: %v", err)
		} else {
			if _, err := io.Copy(io.Discard, comp); err != nil {
				t.Errorf("error reading compressed: %v", err)
			}
			err = comp.Close()
			require.NoError(t, err)
		}
		diffID, err := layers[layerID].DiffID()
		require.NoError(t, err)
		lCache, err = c.Get(diffID)
		require.NoError(t, err)

		digest, err := layers[layerID].Digest()
		lCache, err = c.Get(digest)
		require.NoError(t, err)
	}
}
*/

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

// Compressed at source is not supported
/*
func Test_FSCache_PutAndGet_Compressed_WithZstdCompressionAtSource(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "diskcache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	diskImgFilename := filepath.Join(tempDir, "testdisk.img")
	err = generateRandomFile(diskImgFilename, 8*128*1024)
	require.NoError(t, err)

	c := NewFilesystemCache(tempDir)

	for layerID := 0; layerID < 8; layerID++ {
		layer, err := filesegment.NewLayer(diskImgFilename, filesegment.WithRange(int64(layerID*128*1024), int64((layerID+1)*128*1024)-1))
		require.NoError(t, err)
		lCache, err := c.Put(layer)
		require.NoError(t, err)
		assert.NotNil(t, lCache)

		if comp, err := lCache.Compressed(); err != nil {
			t.Errorf("Compressed: %v", err)
		} else {
			if _, err := io.Copy(io.Discard, comp); err != nil {
				t.Errorf("error reading compressed: %v", err)
			}
			err = comp.Close()
			require.NoError(t, err)
		}
		diffID, err := layer.DiffID()
		require.NoError(t, err)
		lCache, err = c.Get(diffID)
		require.NoError(t, err)

		digest, err := layer.Digest()
		lCache, err = c.Get(digest)
		require.NoError(t, err)
	}
}
*/

func Test_FSCache_PutAndGet_Uncompressed_WithZstdCompressionAtSource(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "diskcache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	diskImgFilename := filepath.Join(tempDir, "testdisk.img")
	err = generateRandomFile(diskImgFilename, 10000)
	require.NoError(t, err)

	c := NewFilesystemCache(tempDir)

	for layerID := 0; layerID < 8; layerID++ {
		layer, err := filesegment.NewLayer(diskImgFilename, filesegment.WithRange(int64(layerID*1024), int64((layerID+1)*1024)))
		require.NoError(t, err)
		lCache, err := c.Put(layer)
		require.NoError(t, err)
		assert.NotNil(t, lCache)

		if comp, err := lCache.Uncompressed(); err != nil {
			t.Errorf("Compressed: %v", err)
		} else {
			if _, err := io.Copy(io.Discard, comp); err != nil {
				t.Errorf("error reading compressed: %v", err)
			}
			err = comp.Close()
			require.NoError(t, err)
		}
		diffID, err := layer.DiffID()
		require.NoError(t, err)
		lCache, err = c.Get(diffID)
		require.NoError(t, err)

		digest, err := layer.Digest()
		lCache, err = c.Get(digest)
		require.NoError(t, err)
	}
}

func corruptFile(filepath string) error {
	f, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	bytesToAppend := []byte{0xBA, 0xAD, 0xF0, 0x00}
	_, err = f.Write(bytesToAppend)
	if err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}
	fmt.Println("File corrupted successfully")
	return nil
}

func TestFscache_Put_GetFromCorruptedFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "diskcache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	c := NewFilesystemCache(tempDir)

	img, err := random.Image(1024*100, 2)
	require.NoError(t, err)

	layers, err := img.Layers()
	require.NoError(t, err)

	lCache, err := c.Put(layers[0])
	require.NoError(t, err)
	assert.NotNil(t, lCache)

	if uncomp, err := lCache.Uncompressed(); err != nil {
		t.Errorf("Compressed: %v", err)
	} else {
		if _, err := io.Copy(io.Discard, uncomp); err != nil {
			t.Errorf("error reading compressed: %v", err)
		}
		err = uncomp.Close()
		require.NoError(t, err)
	}
	diffID, err := layers[0].DiffID()
	require.NoError(t, err)
	lCache, err = c.Get(diffID)
	require.NoError(t, err)
	digest, err := layers[0].Digest()

	err = corruptFile(filepath.Join(tempDir, digest.String()))
	require.NoError(t, err)

	lCache, err = c.Get(diffID)
	require.Error(t, cache.ErrNotFound)
}
