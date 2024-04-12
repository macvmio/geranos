package filesegment

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseIntPair(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedA   int64
		expectedB   int64
		expectError bool
	}{
		{"Valid pair", "10-20", 10, 20, false},
		{"Invalid format", "10-", 0, 0, true},
		{"Non integer", "ten-twenty", 0, 0, true},
		{"Single number", "10", 0, 0, true},
		{"Three numbers", "10-20-30", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, b, err := parseIntPair(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedA, a)
				assert.Equal(t, tt.expectedB, b)
			}
		})
	}
}

func TestParseDescriptor(t *testing.T) {
	descriptor := v1.Descriptor{
		MediaType: MediaType,
		Digest:    v1.Hash{Algorithm: "sha256", Hex: "abc123"},
		Annotations: map[string]string{
			FilenameAnnotationKey: "image.jpg",
			RangeAnnotationKey:    "100-200",
		},
	}

	t.Run("Valid Descriptor", func(t *testing.T) {
		d, err := ParseDescriptor(descriptor)
		assert.NoError(t, err)
		assert.Equal(t, "image.jpg", d.Filename())
		assert.Equal(t, int64(100), d.Start())
		assert.Equal(t, int64(200), d.Stop())
		assert.Equal(t, descriptor.Digest, d.Digest())
	})

	t.Run("Missing Filename", func(t *testing.T) {
		modifiedDescriptor := descriptor
		delete(modifiedDescriptor.Annotations, FilenameAnnotationKey)
		_, err := ParseDescriptor(modifiedDescriptor)
		assert.Error(t, err)
	})

	t.Run("Missing Range", func(t *testing.T) {
		modifiedDescriptor := descriptor
		delete(modifiedDescriptor.Annotations, RangeAnnotationKey)
		_, err := ParseDescriptor(modifiedDescriptor)
		assert.Error(t, err)
	})

	t.Run("Unsupported MediaType", func(t *testing.T) {
		modifiedDescriptor := descriptor
		modifiedDescriptor.MediaType = "unsupported/type"
		_, err := ParseDescriptor(modifiedDescriptor)
		assert.Error(t, err)
	})
}
