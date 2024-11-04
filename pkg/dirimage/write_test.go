package dirimage

import (
	"context"
	"errors"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/macvmio/geranos/pkg/filesegment"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestWrite_ContextCancelledDuringWork(t *testing.T) {

	tempDir, err := os.MkdirTemp("", "write-test-*")
	require.NoErrorf(t, err, "unable to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)

	err = generateRandomFile(filepath.Join(tempDir, "file1.img"), 100)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(tempDir, "testdir1"), 0o777)
	require.NoError(t, err)

	img := empty.Image
	for i := 0; i < 10; i++ {
		layer, err := filesegment.NewLayer(filepath.Join(tempDir, "file1.img"), filesegment.WithRange(int64(i*10), int64(i*10+9)))
		require.NoError(t, err)

		img, err = mutate.Append(img, mutate.Addendum{
			Layer:       layer,
			Annotations: layer.Annotations(),
			MediaType:   "",
		})
		require.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	di, err := Convert(img)
	require.NoError(t, err)
	err = di.Write(ctx, filepath.Join(tempDir, "testdir1"), WithWorkersCount(2))

	if err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("Write did not return expected context.Canceled error during work, got: %v", err)
	}
}

func TestDirImage_deleteManifest(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "test-delete-manifest")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Cleanup after the test

	di := &DirImage{}

	t.Run("FileExists", func(t *testing.T) {
		// Create the manifest file inside the temporary directory
		manifestPath := filepath.Join(tempDir, LocalManifestFilename)
		if err := os.WriteFile(manifestPath, []byte(`{}`), 0644); err != nil {
			t.Fatalf("Failed to create manifest file: %v", err)
		}

		// Call the deleteManifest function
		err := di.deleteManifest(tempDir)
		if err != nil {
			t.Fatalf("Expected no error, but got: %v", err)
		}

		// Check if the file is deleted
		if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
			t.Fatalf("Expected file to be deleted, but it still exists")
		}
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		// Ensure the manifest file does not exist
		manifestPath := filepath.Join(tempDir, LocalManifestFilename)
		if err := os.Remove(manifestPath); !os.IsNotExist(err) && err != nil {
			t.Fatalf("Failed to remove manifest file in setup: %v", err)
		}

		// Call the deleteManifest function
		err := di.deleteManifest(tempDir)
		if err != nil {
			t.Fatalf("Expected no error when file does not exist, but got: %v", err)
		}
	})
}
