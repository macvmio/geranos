package image

import (
	"crypto/rand"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/tomekjarosik/geranos/pkg/image/duplicator"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
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

func TestLayoutMapper_Read(t *testing.T) {
	lm := NewLayoutMapper("testdata")
	ref, err := name.ParseReference("vm1")
	if err != nil {
		t.Fatalf("unable to parse reference: %v", err)
	}
	img, err := lm.Read(ref)
	if err != nil {
		t.Fatalf("unable to read layout from disk: %v", err)
	}
	err = validate.Image(img, validate.Fast)
	if err != nil {
		t.Fatalf("image validation error: %v", err)
	}
}

func TestLayoutMapper_Read_VariousChunkSizes(t *testing.T) {
	hashBefore := hashFromFile(t, "testdata/vm1/disk.blob")
	tempDir, err := os.MkdirTemp("", "oci-test-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	lmSrc := NewLayoutMapper("testdata")
	lmDst := NewLayoutMapper(tempDir)
	srcRef, err := name.ParseReference("vm1")
	if err != nil {
		t.Fatalf("unable to parse source reference: %v", err)
	}
	dstRef, err := name.ParseReference("vmdst")
	if err != nil {
		t.Fatalf("unable to parse destination reference: %v", err)
	}
	for chunkSize := int64(1); chunkSize < 10; chunkSize++ {
		img, err := lmSrc.Read(srcRef, WithChunkSize(chunkSize))
		if err != nil {
			t.Fatalf("unable to read image: %v", err)
		}
		err = validate.Image(img, validate.Fast)
		if err != nil || img == nil {
			t.Fatalf("img is not correct: %v", err)
		}
		err = lmDst.Write(img, dstRef, nil) // TODO: Make progress an Option
		if err != nil {
			t.Fatalf("unable to write image to destination: %v", err)
		}
		hashAfter := hashFromFile(t, filepath.Join(tempDir, dstRef.String(), "disk.blob"))
		if hashBefore != hashAfter {
			t.Fatalf("hashes differ: expected: %v, got: %v", hashBefore, hashAfter)
		}
	}
}

// GenerateRandomFile creates a file of the specified size filled with random bytes
// using io.CopyN for efficient copying.
func generateRandomFile(fileName string, size int64) error {
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

func TestLayoutMapper_Read_MustOptimizeDiskSpace(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "optimized-disk-*")
	if err != nil {
		t.Fatalf("unable to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	testRepoDir := path.Join(tempDir, "oci.jarosik.online/testrepo")
	err = os.MkdirAll(path.Join(testRepoDir, "a:v1"), 0o777)
	if err != nil {
		t.Fatalf("unable to create directory: %v", err)
		return
	}
	optimalRepoDir := path.Join(tempDir, "oci.jarosik.online/optimalrepo")
	err = os.MkdirAll(path.Join(optimalRepoDir, "a:v1"), 0o777)
	if err != nil {
		t.Fatalf("unable to create directory: %v", err)
		return
	}

	MB := int64(1024 * 1024)
	chunkSize := MB
	err = generateRandomFile(path.Join(testRepoDir, "a:v1/disk.img"), 32*MB)
	if err != nil {
		t.Fatalf("unable to generate file: %v", err)
		return
	}
	srcRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v1"))
	if err != nil {
		t.Fatalf("unable to parse reference: %v", err)
		return
	}
	lm := NewLayoutMapper(tempDir)
	img1, err := lm.Read(srcRef, WithChunkSize(chunkSize))
	if err != nil {
		t.Fatalf("unable to read disk image: %v", err)
		return
	}
	for i := 2; i < 12; i++ {
		r, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v%d", i))
		if err != nil {
			t.Fatalf("unable to parse reference %d: %v", i, err)
		}
		err = lm.Write(img1, r, nil)
		if err != nil {
			t.Fatalf("unable to write image %d: %v", i, err)
			return
		}
		err = duplicator.CloneDirectory(path.Join(testRepoDir, "a:v1"), path.Join(optimalRepoDir, fmt.Sprintf("a:v%d", i)), false)
		if err != nil {
			t.Fatalf("unable to clone directory: %v", err)
		}
	}
	for _, repo := range []string{testRepoDir, optimalRepoDir} {
		diskUsage, err := directoryDiskUsage(testRepoDir)
		if err != nil {
			t.Fatalf("unable to calculate disk usage: %v", err)
		}
		fmt.Printf("[%v] total disk used: %v\n", repo, diskUsage)
	}

}
