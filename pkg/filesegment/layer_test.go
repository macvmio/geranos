package filesegment

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/require"
	"runtime"
	"testing"
)

// NOTE: layout.Write(tempDir,ii) ->
//
//	   There is a renamer function:
//		  if renamer != nil {
//			open = func() (*os.File, error) { return os.CreateTemp(dir, hash.Hex) }
//		  }
//
// The first problem is that it can write to same temporary file, which could be fixed by ' return os.CreateTemp(dir, hash.Hex + "*")
// The second problem is on Windows, that renamer can't rename a file if similar file already exists
// os.Rename() is not a atomic operation on Windows
// This happens because layers have the same hash (which possible if layers have all zeroes)
func TestLayer_AppendLayers(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping because layout.Write does not work correctly for same layers on Windows")
	}

	img := empty.Image
	img = mutate.MediaType(img, types.OCIManifestSchema1)
	img = mutate.ConfigMediaType(img, types.OCIConfigJSON)

	tempDir := t.TempDir()
	for i := 0; i < 8; i++ {
		layer, err := NewLayer("testdata/disk.img", WithRange(int64(i*10), int64(i*10+9)))
		require.NoError(t, err)
		img, err = mutate.AppendLayers(img, layer)
		if err != nil {
			t.Errorf("unable to append layer")
		}
	}
	img, err := mutate.Config(img, v1.Config{})
	require.NoError(t, err)

	var ii v1.ImageIndex
	ii = empty.Index
	ii = mutate.AppendManifests(ii, mutate.IndexAddendum{
		Add: img,
	})
	_, err = layout.Write(tempDir, ii)
	require.NoError(t, err)
}

// This works compared to previous test because each layer has a different hash
func TestLayer_AppendLayersOptimistic(t *testing.T) {
	img := empty.Image
	img = mutate.MediaType(img, types.OCIManifestSchema1)
	img = mutate.ConfigMediaType(img, types.OCIConfigJSON)

	tempDir := t.TempDir()
	for i := 0; i < 8; i++ {
		layer, err := NewLayer("testdata/disk.img", WithRange(int64(i*9), int64(i*10+9)))
		require.NoError(t, err)
		img, err = mutate.AppendLayers(img, layer)
		if err != nil {
			t.Errorf("unable to append layer")
		}
	}
	img, err := mutate.Config(img, v1.Config{})
	require.NoError(t, err)

	var ii v1.ImageIndex
	ii = empty.Index
	ii = mutate.AppendManifests(ii, mutate.IndexAddendum{
		Add: img,
	})
	_, err = layout.Write(tempDir, ii)
	require.NoError(t, err)
}

func TestLayer_Functionality(t *testing.T) {
	// Assuming testdata/disk.img exists and has content
	layerFile := "testdata/disk.img"
	layer, err := NewLayer(layerFile)
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
	if err != nil || mediaType != MediaType {
		t.Errorf("MediaType failed or returned unexpected value: %v, %v", err, mediaType)
	}

	// Testing Size
	_, err = layer.Size()
	if err != nil {
		t.Errorf("Size failed: %v", err)
	}

	// Testing Append to empty Image
	img := empty.Image
	_, err = mutate.AppendLayers(img, layer)
	if err != nil {
		t.Errorf("unable to append layer: %v", err)
	}
}
