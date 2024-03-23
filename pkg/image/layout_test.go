package image

import (
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"log"
	"os"
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

func TestLayoutMapper_Read_VariouChunkSizes(t *testing.T) {
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
