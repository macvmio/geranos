package sparsefile

import (
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path"
	"testing"
)

func TestOverwrite(t *testing.T) {
	tests := []struct {
		name        string
		dstInitial  string // Initial content of dst
		src         string // Content to overwrite dst with
		srcBufSize  int
		dstBufSize  int
		wantWritten int64  // Expected number of bytes actually written
		wantSkipped int64  // Expected number of bytes skipped because they are the same
		wantFinal   string // Expected final content of dst
		wantErr     bool   // Whether an error is expected
	}{
		{
			name:        "overwrite first chunk",
			dstInitial:  "Hello, World!",
			src:         "Greetings!",
			srcBufSize:  8,
			dstBufSize:  8,
			wantFinal:   "Greetings!ld!",
			wantWritten: 10,
			wantSkipped: 0,

			wantErr: false,
		},
		{
			name:        "Partial overwrite, some content same",
			dstInitial:  "Hello,  World!",
			src:         "Hello,  Go!",
			wantFinal:   "Hello,  Go!ld!",
			srcBufSize:  8,
			dstBufSize:  8,
			wantWritten: 3,
			wantSkipped: 8, // "Hello,  " is the same

			wantErr: false,
		},
		{
			name:        "Complete match, all skipped",
			dstInitial:  "Hello, World!",
			src:         "Hello, World!",
			srcBufSize:  8,
			dstBufSize:  8,
			wantWritten: 0,
			wantSkipped: 13,
			wantFinal:   "Hello, World!",
			wantErr:     false,
		},
		{
			name:        "Complete match, dst buffer larger",
			dstInitial:  "Hello, World!",
			src:         "Hello, World!",
			srcBufSize:  8,
			dstBufSize:  10,
			wantWritten: 0,
			wantSkipped: 13,
			wantFinal:   "Hello, World!",
			wantErr:     false,
		},
		{
			name:        "Partial match in the middle",
			dstInitial:  "Hello, W12345678World!",
			src:         "123456781234567812345678",
			srcBufSize:  8,
			dstBufSize:  10,
			wantWritten: 16,
			wantSkipped: 8,
			wantFinal:   "123456781234567812345678",
			wantErr:     false,
		},
		{
			name:        "Partial match at the end",
			dstInitial:  "Hello, W123456",
			src:         "12345678123456",
			srcBufSize:  8,
			dstBufSize:  30,
			wantWritten: 8,
			wantSkipped: 6,
			wantFinal:   "12345678123456",
			wantErr:     false,
		},
		{
			name:        "dst initial is empty",
			dstInitial:  "",
			src:         "12345678123456",
			wantFinal:   "12345678123456",
			srcBufSize:  8,
			dstBufSize:  30,
			wantWritten: 14,
			wantSkipped: 0,

			wantErr: false,
		},
		{
			name:        "dst initial is larger than src",
			dstInitial:  "000000000000000000000000000",
			src:         "12345678123456",
			wantFinal:   "123456781234560000000000000",
			srcBufSize:  8,
			dstBufSize:  30,
			wantWritten: 14,
			wantSkipped: 0,

			wantErr: false,
		},
		{
			name:        "input is multiplier of src buffer",
			dstInitial:  "1234567812345678",
			src:         "1234567812345678",
			wantFinal:   "1234567812345678",
			srcBufSize:  8,
			dstBufSize:  8,
			wantWritten: 0,
			wantSkipped: 16,

			wantErr: false,
		},
	}

	createFileWithContent := func(t *testing.T, name string, content string) (f *os.File, closer func()) {
		t.Helper()

		f, err := os.Create(name)
		require.NoError(t, err)
		_, err = io.WriteString(f, content)
		require.NoError(t, err)
		_, err = f.Seek(0, io.SeekStart)
		require.NoError(t, err)
		return f, func() {
			require.NoError(t, f.Close())
		}
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			src, closerSrc := createFileWithContent(t, path.Join(tmp, "src.tmp"), tt.src)
			dst, closerDst := createFileWithContent(t, path.Join(tmp, "dst.tmp"), tt.dstInitial)
			defer closerDst()
			defer closerSrc()

			written, skipped, err := overwriteBuffer(dst, src, make([]byte, tt.srcBufSize), make([]byte, tt.dstBufSize))
			if (err != nil) != tt.wantErr {
				t.Errorf("Overwrite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check the final content of dst
			_, err = dst.Seek(0, io.SeekStart)
			require.NoError(t, err)
			finalDstContent, err := io.ReadAll(dst)
			require.NoError(t, err)
			if string(finalDstContent) != tt.wantFinal {
				t.Errorf("Final dst content = %v, want %v", string(finalDstContent), tt.wantFinal)
			}
			if written != tt.wantWritten {
				t.Errorf("Overwrite() written = %v, want %v", written, tt.wantWritten)
			}
			if skipped != tt.wantSkipped {
				t.Errorf("Overwrite() skipped = %v, want %v", skipped, tt.wantSkipped)
			}
		})
	}
}
