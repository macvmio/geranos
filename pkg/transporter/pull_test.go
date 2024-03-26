package transporter

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestPullAndPush(t *testing.T) {
	s := httptest.NewServer(prepareRegistry())
	defer s.Close()

	tempDir, opts := optionsForTesting(t)

	ref := refOnServer(s.URL, "test-vm:1.0")

	err := Pull(ref, opts...)
	assert.ErrorContains(t, err, "NAME_UNKNOWN: Unknown name")

	shaBefore := makeTestVMAt(t, tempDir, ref)
	err = Push(ref, opts...)
	assert.NoError(t, err)

	deleteTestVMAt(t, tempDir, ref)

	err = Pull(ref, opts...)
	assert.NoError(t, err)

	shaAfter := hashFromFile(t, filepath.Join(tempDir, "images", ref, "disk.img"))
	assert.Equal(t, shaBefore, shaAfter)

	fmt.Println("expected cache:")
	err = Pull(ref, opts...)
	assert.NoError(t, err)
	shaAfter = hashFromFile(t, filepath.Join(tempDir, "images", ref, "disk.img"))
	assert.Equal(t, shaBefore, shaAfter)
}
