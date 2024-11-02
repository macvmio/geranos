package filesegment

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPartialFileReader(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a temporary file with known content
	filePath := filepath.Join(tempDir, "testfile.txt")
	content := []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ") // 26 bytes
	err := os.WriteFile(filePath, content, 0644)
	require.NoError(t, err, "Failed to write test file")

	// Define test cases with start and stop positions
	testCases := []struct {
		name        string
		start       int64
		stop        int64
		expected    []byte
		expectError bool
	}{
		{
			name:     "First 10 bytes",
			start:    0,
			stop:     9,
			expected: []byte("ABCDEFGHIJ"),
		},
		{
			name:     "Middle 5 bytes",
			start:    10,
			stop:     14,
			expected: []byte("KLMNO"),
		},
		{
			name:     "Last 6 bytes",
			start:    20,
			stop:     25,
			expected: []byte("UVWXYZ"),
		},
		{
			name:     "Entire file",
			start:    0,
			stop:     25,
			expected: content,
		},
		{
			name:        "Invalid range (start > stop)",
			start:       15,
			stop:        10,
			expectError: true,
		},
		{
			name:     "Range beyond file size",
			start:    24,
			stop:     30,
			expected: []byte("YZ"),
		},
		{
			name:        "Start beyond file size",
			start:       30,
			stop:        35,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Create a new partialFileReader
			pfr, err := newPartialFileReader(filePath, tc.start, tc.stop)
			if tc.expectError {
				require.Error(t, err, "Expected error for invalid range")
				return
			}
			require.NoError(t, err, "Failed to create partialFileReader")
			defer pfr.Close()

			// Read data from partialFileReader
			var result []byte
			buf := make([]byte, 5) // Read in chunks of 5 bytes
			for {
				n, err := pfr.Read(buf)
				if n > 0 {
					result = append(result, buf[:n]...)
				}
				if err == io.EOF {
					break
				}
				require.NoError(t, err, "Error reading from partialFileReader")
			}

			require.Equal(t, tc.expected, result, "Read data does not match expected")
		})
	}
}
