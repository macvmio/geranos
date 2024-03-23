package segmentlayer

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"os"
	"testing"
)

func TestLayer_AppendLayers(t *testing.T) {
	img := empty.Image
	tempDir, err := os.MkdirTemp("", "test-image-*")
	if err != nil {
		t.Fatalf("unable to create temp directory, got %v", err)
	}
	defer os.RemoveAll(tempDir)

	for i := 0; i < 4; i++ {
		layer1, err := FromFile("testdata/disk.img", WithRange(int64(i*10), int64(i*10+9)))
		if err != nil {
			t.Errorf("unable to create layer out of input file from range 0-8")
		}
		img, err = mutate.AppendLayers(img, layer1)
		if err != nil {
			t.Errorf("unable to append layer1")
		}
	}
	img = mutate.MediaType(img, "vnd.tomekjarosik.geranos.image.distribution.v1")
	img = mutate.ConfigMediaType(img, "vnd.tomekjarosik.geranos.image.distribution.v1")

	rawConfig, err := img.RawConfigFile()
	if err != nil {
		t.Errorf("unable to read raw config file, got %v", err)
	}
	fmt.Printf("rawConfig=%v\n", string(rawConfig))
	manifest, err := img.RawManifest()
	if err != nil {
		t.Errorf("unable to read raw manifest, got %v", err)
	}
	fmt.Printf("rawManifest=%v\n", string(manifest))

	ii := empty.Index
	_, err = layout.Write(tempDir, ii)
	l1, err := layout.FromPath(tempDir)
	if err != nil {
		t.Errorf("unable to load layout from path, got %v", err)
	}
	err = l1.AppendImage(img)
	if err != nil {
		t.Errorf("append image failed, got %v", err)
	}
}

func TestLayer_Functionality(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test-image-*")
	if err != nil {
		t.Fatalf("unable to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Assuming testdata/disk.img exists and has content
	layerFile := "testdata/disk.img"
	layer, err := FromFile(layerFile)
	if err != nil {
		t.Fatalf("unable to create layer from file %s: %v", layerFile, err)
	}

	// Testing Uncompressed
	_, err = layer.Uncompressed()
	if err != nil {
		t.Errorf("Uncompressed failed: %v", err)
	}

	// Testing Compressed
	_, err = layer.Compressed()
	if err != nil {
		t.Errorf("Compressed failed: %v", err)
	}

	// Testing Digest
	_, err = layer.Digest()
	if err != nil {
		t.Errorf("Digest failed: %v", err)
	}

	// Testing DiffID
	_, err = layer.DiffID()
	if err != nil {
		t.Errorf("DiffID failed: %v", err)
	}

	// Testing MediaType
	mediaType, err := layer.MediaType()
	if err != nil || mediaType != FileSegmentMediaType {
		t.Errorf("MediaType failed or returned unexpected value: %v, %v", err, mediaType)
	}

	// Testing Size
	_, err = layer.Size()
	if err != nil {
		t.Errorf("Size failed: %v", err)
	}

	// Testing Append to empty Image
	img := empty.Image
	img, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Errorf("unable to append layer: %v", err)
	}
}
