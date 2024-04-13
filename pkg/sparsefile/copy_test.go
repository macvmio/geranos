package sparsefile

import (
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/magiconair/properties/assert"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestCopy_SparseBasic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sparsefile-test-*")
	if err != nil {
		t.Errorf("unable to create temp dir, got %v", tempDir)
	}
	defer os.RemoveAll(tempDir)

	testFilename := filepath.Join(tempDir, "test.bin")
	f, err := os.Create(testFilename)
	if err != nil {
		t.Errorf("unable to create test file, got %v", err)
	}
	defer f.Close()
	_, err = f.Write([]byte("start"))
	if err != nil {
		t.Errorf("unable to write to a test file, got %v", err)
	}
	zeroBytes := make([]byte, 0)
	for i := 0; i < 1024; i++ {
		zeroBytes = append(zeroBytes, 0)
	}
	for i := 0; i < 20000; i++ {
		n, err := f.Write(zeroBytes)
		if err != nil {
			t.Errorf("unable to write zeroes, got %v", err)
		}
		if n != 1024 {
			t.Errorf("unexpected short write, expected 1024, got %v", n)
		}
	}
	n, err := f.Write([]byte("end"))
	if err != nil {
		t.Errorf("unable to write, got %v", err)
	}
	if n != 3 {
		t.Errorf("write too short: expected 3 bytes, got %v", n)
	}
	inputFile, err := os.Open(testFilename)
	if err != nil {
		t.Errorf("unable to open test file, got %v", err)
	}
	defer inputFile.Close()

	sparseFilename := filepath.Join(tempDir, "sparsefile.bin")
	sparseFile, err := os.Create(sparseFilename)
	if err != nil {
		t.Errorf("unable to create sparse file, got %v", err)
	}

	written, skipped, err := Copy(sparseFile, inputFile)
	if err != nil {
		t.Errorf("unable to copy input file to sparse file, got %v", err)
	}
	if written+skipped != 20480008 {
		t.Errorf("invalid number of written+skipped bytes, expected 20480008, got %v", written+skipped)
	}

	cmd := exec.Command("du", "-h", sparseFilename)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("unable to execute command '%v'", cmd.String())
	}
	fmt.Println(string(outBytes))
	if !strings.Contains(string(outBytes), "100K") {
		t.Errorf("not a sparse file, invalid size")
	}

	inputFileHash := hashFromFile(t, testFilename)
	outputFileHash := hashFromFile(t, sparseFilename)
	if inputFileHash != outputFileHash {
		t.Errorf("hashes do not match")
	}
}

func TestCopy_EmptySuffix(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sparsefile-test-*")
	if err != nil {
		t.Errorf("unable to create temp dir, got %v", tempDir)
	}
	defer os.RemoveAll(tempDir)

	testFilename := filepath.Join(tempDir, "test.bin")
	f, err := os.Create(testFilename)
	if err != nil {
		t.Errorf("unable to create test file, got %v", err)
	}
	defer f.Close()
	_, err = f.Write([]byte("start"))
	if err != nil {
		t.Errorf("unable to write to a test file, got %v", err)
	}
	zeroBytes := make([]byte, 0)
	for i := 0; i < 1024; i++ {
		zeroBytes = append(zeroBytes, 0)
	}
	for i := 0; i < 20000; i++ {
		n, err := f.Write(zeroBytes)
		if err != nil {
			t.Errorf("unable to write zeroes, got %v", err)
		}
		if n != 1024 {
			t.Errorf("unexpected short write, expected 1024, got %v", n)
		}
	}
	err = f.Close()
	if err != nil {
		t.Errorf("unable to close a file, got %v", err)
	}

	inputFile, err := os.Open(testFilename)
	defer inputFile.Close()

	sparseFilename := filepath.Join(tempDir, "sparsefiletest.bin")
	sparseFile, err := os.Create(sparseFilename)
	if err != nil {
		t.Errorf("unable to create sparse file, got %v", err)
	}

	written, skipped, err := Copy(sparseFile, inputFile)
	if err != nil {
		t.Errorf("error while copying to a sparse file, got %v", err)
	}
	assert.Equal(t, int64(20480005), written+skipped)
	sparseFile.Close()

	cmd := exec.Command("du", "-h", sparseFilename)
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("unable to execute command '%v', got %v", cmd.String(), err)
	}
	fmt.Println(string(outBytes))
	if !strings.Contains(string(outBytes), "68K") {
		t.Errorf("invalid size of sparse file")
	}

	cmd = exec.Command("ls", "-lah", sparseFilename)
	outBytes, err = cmd.CombinedOutput()
	if err != nil {
		t.Errorf("unable to execute command '%v', got %v", cmd.String(), err)
	}
	fmt.Println(string(outBytes))
	if !strings.Contains(string(outBytes), "20M") {
		t.Errorf("invalid size of sparse file")
	}

	inputFileHash := hashFromFile(t, testFilename)
	outputFileHash := hashFromFile(t, sparseFilename)

	if inputFileHash != outputFileHash {
		t.Errorf("hash mismatch")
	}
}
