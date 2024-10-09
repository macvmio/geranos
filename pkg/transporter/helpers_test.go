package transporter

import (
	"crypto/rand"
	"fmt"
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// prepareRegistryWithLogging wraps the original prepareRegistry function
// to add logging of requests.
func prepareRegistryWithRecorder(rec *[]http.Request) http.Handler {
	originalHandler := prepareRegistry() // Assuming this is your function that sets up the registry
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*rec = append(*rec, *r)
		// Call the original handler
		originalHandler.ServeHTTP(w, r)
	})
}

func prepareRegistry() http.Handler {
	registryOpts := []registry.Option{registry.WithReferrersSupport(true)}
	return registry.New(registryOpts...)
}

func optionsForTesting(t *testing.T) (tempDir string, opts []Option) {
	var err error
	tempDir, err = os.MkdirTemp("", "geranos-test-*")
	if err != nil {
		t.Fatalf("unable to create temp dir: %v", err)
	}
	opts = []Option{
		WithImagesPath(filepath.Join(tempDir, "images")),
		WithCachePath(filepath.Join(tempDir, "cache")),
	}
	return tempDir, opts
}

func refOnServer(serverUrl string, repository string) string {
	return strings.TrimPrefix(serverUrl, "http://") + "/" + repository
}

func makeFileAt(t *testing.T, filename, content string) {
	f, err := os.Create(filename)
	assert.NoError(t, err)
	defer f.Close()
	_, err = f.Write([]byte(content))
	assert.NoError(t, err)
}

func makeTestVMAt(t *testing.T, tempDir, ref string) (sha string) {
	t.Helper()
	d := filepath.Join(tempDir, "images", ref)
	err := os.MkdirAll(d, os.ModePerm)
	assert.NoError(t, err)
	require.NoError(t, err)
	makeFileAt(t, filepath.Join(d, "disk.img"), "some fake image data")
	makeFileAt(t, filepath.Join(d, "config.json"), `{"disk_size"": 123}`)

	return hashFromFile(t, filepath.Join(d, "disk.img"))
}

func makeTestVMWithContent(t *testing.T, tempDir, ref string, content string) (sha string) {
	t.Helper()
	d := filepath.Join(tempDir, "images", ref)
	err := os.MkdirAll(d, os.ModePerm)
	assert.NoError(t, err)
	require.NoError(t, err)
	makeFileAt(t, filepath.Join(d, "disk.img"), content)

	return hashFromFile(t, filepath.Join(d, "disk.img"))
}

func makeRandomFile(t *testing.T, fileName string, size int64) error {
	t.Helper()
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

func makeBigTestVMAt(t *testing.T, tempDir, ref string) (sha string) {
	t.Helper()
	d := filepath.Join(tempDir, "images", ref)
	err := os.MkdirAll(d, os.ModePerm)
	assert.NoError(t, err)
	require.NoError(t, err)
	err = makeRandomFile(t, filepath.Join(d, "disk.img"), 270*1024*1024)
	require.NoError(t, err)

	return hashFromFile(t, filepath.Join(d, "disk.img"))
}

func modifyByteInFileToEnsureDifferent(t *testing.T, filePath string, offset int64) {
	t.Helper()

	// Open the file for reading and writing. The file must exist.
	f, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	require.NoError(t, err, "opening file failed")
	defer f.Close()

	// Seek to the position, offset bytes from the end of the file.
	_, err = f.Seek(-offset, io.SeekEnd)
	require.NoError(t, err, "seeking in file failed")

	// Read the current byte value at the position.
	originalByte := make([]byte, 1)
	_, err = f.Read(originalByte)
	require.NoError(t, err, "reading from file failed")

	newValue := originalByte[0] + 1
	if newValue == originalByte[0] {
		newValue = originalByte[0] - 1
	}

	// Now, seek back to the original position to overwrite the byte.
	_, err = f.Seek(-1, io.SeekCurrent)
	require.NoError(t, err, "seeking back in file failed")

	// Write the new (different) byte value back to the file.
	_, err = f.Write([]byte{newValue})
	require.NoError(t, err, "writing to file failed")

	// Optionally, you might want to ensure the file changes have been flushed to disk.
	err = f.Sync()
	require.NoError(t, err, "syncing file failed")
}

func modifyBigTestVMAt(t *testing.T, tempDir, ref string, offset int64) (sha string) {
	t.Helper()
	d := filepath.Join(tempDir, "images", ref)
	err := os.MkdirAll(d, os.ModePerm)
	assert.NoError(t, err)
	require.NoError(t, err)
	modifyByteInFileToEnsureDifferent(t, filepath.Join(d, "disk.img"), offset)
	require.NoError(t, err)

	return hashFromFile(t, filepath.Join(d, "disk.img"))
}

func deleteTestVMAt(t *testing.T, tempDir, ref string) {
	d := filepath.Join(tempDir, "images", ref)
	err := os.RemoveAll(d)
	assert.NoError(t, err)
}

func hashFromFile(t *testing.T, filename string) string {
	f, err := os.Open(filename)
	if err != nil {
		t.Errorf("unexpected error while opening a file, got %v", err)
	}
	defer f.Close()
	h, _, err := v1.SHA256(f)
	if err != nil {
		t.Errorf("unable to calculate SHA256, got %v", err)
	}
	return h.Hex
}
