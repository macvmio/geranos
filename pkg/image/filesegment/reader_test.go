package filesegment

import (
	"errors"
	"io"
	"os"
	"testing"
)

func TestPartialFileReaderOpen(t *testing.T) {
	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatalf("unable to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	_, err = tmpfile.WriteString("Hello, world")
	if err != nil {
		t.Fatalf("unable to write to temporary file: %v", err)
	}
	tmpfile.Close()

	// Test opening the file
	pfr := partialFileReader{
		filePath: tmpfile.Name(),
		start:    0,
		stop:     int64(len("Hello, world")),
	}

	err = pfr.open()
	if err != nil {
		t.Errorf("open failed: %v", err)
	}

	// Ensure file is not nil
	if pfr.f == nil {
		t.Errorf("file was not opened")
	}
}

func TestPartialFileReaderRead(t *testing.T) {
	content := "Hello, world"

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatalf("unable to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	_, err = tmpfile.WriteString(content)
	if err != nil {
		t.Fatalf("unable to write to temporary file: %v", err)
	}
	err = tmpfile.Close()
	if err != nil {
		t.Errorf("unable to clse tmpfile")
	}

	// Instantiate partialFileReader
	pfr := partialFileReader{
		filePath: tmpfile.Name(),
		start:    0,
		stop:     int64(len(content)),
	}

	buffer := make([]byte, len(content))
	n, err := pfr.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Errorf("Read failed: %v", err)
	}

	if n != len(content) {
		t.Errorf("Expected to read %d bytes, read %d", len(content), n)
	}

	if string(buffer) != content {
		t.Errorf("Expected content %q, got %q", content, string(buffer))
	}
}

// TestPartialFileReadOnPartialContent checks if it can read a portion of the file correctly.
func TestPartialFileReadOnPartialContent(t *testing.T) {
	content := "Hello, Go test world!"
	partContent := "Go test"

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatalf("unable to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	_, err = tmpfile.WriteString(content)
	if err != nil {
		t.Fatalf("unable to write to temporary file: %v", err)
	}
	tmpfile.Close()

	// Instantiate partialFileReader for partial content
	pfr := partialFileReader{
		filePath: tmpfile.Name(),
		start:    int64(len("Hello, ")),
		stop:     int64(len("Hello, ") + len(partContent) - 1),
	}

	buffer := make([]byte, len(partContent))
	n, err := pfr.Read(buffer)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Errorf("Read failed: %v", err)
	}

	if n != len(partContent) {
		t.Errorf("Expected to read %d bytes, read %d", len(partContent), n)
	}

	if string(buffer[:n]) != partContent {
		t.Errorf("Expected content %q, got %q", partContent, string(buffer[:n]))
	}
}

// TestPartialFileReaderEmptyFile tests reading from an empty file.
func TestPartialFileReaderEmptyFile(t *testing.T) {
	// Create a temporary, empty file
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatalf("unable to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	// Instantiate partialFileReader for the empty file
	pfr := partialFileReader{
		filePath: tmpfile.Name(),
		start:    0,
		stop:     0,
	}

	buffer := make([]byte, 10) // buffer size larger than file content
	n, err := pfr.Read(buffer)
	if err != io.EOF {
		t.Errorf("Expected EOF for empty file read, got: %v", err)
	}

	if n != 0 {
		t.Errorf("Expected to read 0 bytes, read %d", n)
	}
}

// TestReadingBeyondStopPosition tests attempting to read beyond the 'stop' position.
func TestReadingBeyondStopPosition(t *testing.T) {
	content := "Hello, Go!"

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatalf("unable to create temporary file: %v", err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	_, err = tmpfile.WriteString(content)
	if err != nil {
		t.Fatalf("unable to write to temporary file: %v", err)
	}
	tmpfile.Close()

	// Instantiate partialFileReader with 'stop' before the end of content
	pfr := partialFileReader{
		filePath: tmpfile.Name(),
		start:    0,
		stop:     int64(len("Hello,")) - 1, // Should stop before " Go!"
	}

	buffer := make([]byte, len(content)) // buffer large enough to potentially read beyond 'stop'
	n, err := pfr.Read(buffer)
	if err != io.EOF {
		t.Errorf("Expected EOF when reading beyond stop, got: %v", err)
	}

	if string(buffer[:n]) != "Hello," {
		t.Errorf("Expected to read up to 'stop' position, got: %q", string(buffer[:n]))
	}
}
