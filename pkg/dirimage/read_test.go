package dirimage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

// Helper function to write a manifest to a directory
func writeManifest(t *testing.T, dir string, manifest *v1.Manifest) {
	t.Helper()
	data, err := json.Marshal(manifest)
	assert.NoError(t, err)

	err = os.WriteFile(filepath.Join(dir, LocalManifestFilename), data, 0644)
	assert.NoError(t, err)
}

func mustHash(t *testing.T, s string) v1.Hash {
	h, err := v1.NewHash(s)
	require.NoError(t, err)
	return h
}

func TestReadDigest_Success(t *testing.T) {
	// Create a temporary directory for the test
	dir, err := os.MkdirTemp("", "test-manifest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	// Create a sample v1.Manifest object
	manifest := &v1.Manifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.docker.distribution.manifest.v2+json",
		Config: v1.Descriptor{
			MediaType: "application/vnd.docker.container.image.v1+json",
			Size:      1234,
			Digest:    mustHash(t, "sha256:1234567812345678123456781234567812345678123456781234567812345678"),
		},
		Layers: []v1.Descriptor{
			{
				MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
				Size:      5678,
				Digest:    mustHash(t, "sha256:1234567812345678123456781234567812345678123456781234567812345678"),
			},
		},
	}

	// Write the manifest to the directory
	writeManifest(t, dir, manifest)

	// Read the digest
	digest, err := ReadDigest(dir)
	assert.NoError(t, err)

	// Manually calculate the expected hash
	jsonData, err := json.Marshal(manifest)
	assert.NoError(t, err)
	expectedHash := sha256.Sum256(jsonData)
	expectedDigest, err := v1.NewHash(fmt.Sprintf("sha256:%x", expectedHash[:]))
	assert.NoError(t, err)

	// Compare the calculated digest with the expected digest
	assert.Equal(t, expectedDigest, digest)
}

func TestReadDigest_ManifestNotFound(t *testing.T) {
	// Create a temporary directory for the test
	dir, err := os.MkdirTemp("", "test-manifest-not-found")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	// Try reading the digest from a directory without a manifest
	_, err = ReadDigest(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to read manifest")
}

func TestReadDigest_InvalidManifest(t *testing.T) {
	// Create a temporary directory for the test
	dir, err := os.MkdirTemp("", "test-invalid-manifest")
	assert.NoError(t, err)
	defer os.RemoveAll(dir)

	// Write invalid manifest data
	err = os.WriteFile(filepath.Join(dir, LocalManifestFilename), []byte("invalid json"), 0644)
	assert.NoError(t, err)

	// Try reading the digest
	_, err = ReadDigest(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid character 'i' looking for beginning of value")
}
