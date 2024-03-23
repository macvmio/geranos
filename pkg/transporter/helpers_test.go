package transporter

import (
	"github.com/google/go-containerregistry/pkg/registry"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	d := filepath.Join(tempDir, "images", ref)
	err := os.MkdirAll(d, 0o777)
	assert.NoError(t, err)
	makeFileAt(t, filepath.Join(d, "disk.img"), "some fake image data")
	makeFileAt(t, filepath.Join(d, "config.json"), `{"disk_size"": 123}`)

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
