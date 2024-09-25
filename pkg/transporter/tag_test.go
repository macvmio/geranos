package transporter

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRetagRemotely_Success(t *testing.T) {
	recordedRequests := make([]http.Request, 0)
	s := httptest.NewServer(prepareRegistryWithRecorder(&recordedRequests))
	defer s.Close()

	// Setup temporary directory and options for testing
	tempDir, opts := optionsForTesting(t)

	// Original reference and new tag reference
	oldRef := refOnServer(s.URL, "test-vm:1.0")
	newRef := refOnServer(s.URL, "test-vm:latest")

	// Step 1: Push an image with the original tag
	makeTestVMAt(t, tempDir, oldRef)
	err := Push(oldRef, opts...)
	require.NoError(t, err)

	// Step 2: Re-tag the image remotely (RetagRemotely)
	err = RetagRemotely(oldRef, newRef, opts...)
	require.NoError(t, err)

	// Step 3: Verify that the re-tagged image exists by pulling it
	err = Pull(newRef, opts...)
	require.NoError(t, err)

	assert.Equal(t, 2, calculateAccessed(recordedRequests, "GET", "/manifests"))
	assert.Equal(t, 1, calculateAccessed(recordedRequests, "PUT", "/v2/test-vm/manifests/1.0"))
	assert.Equal(t, 1, calculateAccessed(recordedRequests, "PUT", "/v2/test-vm/manifests/latest"))
}

func TestRetagRemotely_InvalidOldRef(t *testing.T) {
	_, opts := optionsForTesting(t)

	// Invalid old image reference
	oldRef := "invalid reference"
	newRef := "test-vm:latest"

	err := RetagRemotely(oldRef, newRef, opts...)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse old reference")
}

func TestRetagRemotely_InvalidNewRef(t *testing.T) {
	recordedRequests := make([]http.Request, 0)
	s := httptest.NewServer(prepareRegistryWithRecorder(&recordedRequests))
	defer s.Close()

	tempDir, opts := optionsForTesting(t)

	// Valid old image reference but invalid new reference
	oldRef := refOnServer(s.URL, "test-vm:1.0")
	newRef := "invalid reference"

	makeTestVMAt(t, tempDir, oldRef)
	err := Push(oldRef, opts...)
	require.NoError(t, err)

	err = RetagRemotely(oldRef, newRef, opts...)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to parse new reference")
}

func TestRetagRemotely_FailToFetchImage(t *testing.T) {
	recordedRequests := make([]http.Request, 0)
	s := httptest.NewServer(prepareRegistryWithRecorder(&recordedRequests))
	defer s.Close()

	_, opts := optionsForTesting(t)

	// Valid new reference but old reference that doesn't exist
	oldRef := refOnServer(s.URL, "nonexistent-vm:1.0")
	newRef := refOnServer(s.URL, "test-vm:latest")

	err := RetagRemotely(oldRef, newRef, opts...)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unable to fetch image from registry")
}
