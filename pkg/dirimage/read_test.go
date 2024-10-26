package dirimage

import (
	"context"
	"encoding/json"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/macvmio/geranos/pkg/filesegment"
)

// TestPrepareConfigFile_NewConfig tests prepareConfigFile when the config file does not exist.
func TestPrepareConfigFile_NewConfig(t *testing.T) {
	dir := t.TempDir() // Create a temporary directory
	cfg, err := prepareConfigFile(dir, false)
	if err != nil {
		t.Fatalf("prepareConfigFile returned error: %v", err)
	}

	if cfg.Container != "geranos" {
		t.Errorf("Expected Container to be 'geranos', got '%s'", cfg.Container)
	}

	if cfg.Created.IsZero() {
		t.Errorf("Expected Created time to be set, got zero value")
	}

	labels := cfg.Config.Labels
	if labels == nil || labels["org.opencontainers.image.title"] != "geranos" {
		t.Errorf("Expected label 'org.opencontainers.image.title' to be 'geranos', got '%v'", labels)
	}
}

// TestPrepareConfigFile_ExistingConfig tests prepareConfigFile when a config file exists.
func TestPrepareConfigFile_ExistingConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, LocalConfigFilename)

	// Create a sample config file
	cfgData := `{
        "container": "test_container",
        "created": "2023-10-04T12:00:00Z",
        "config": {
            "Labels": {
                "org.opencontainers.image.title": "test_title"
            }
        }
    }`

	err := os.WriteFile(cfgPath, []byte(cfgData), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := prepareConfigFile(dir, false)
	if err != nil {
		t.Fatalf("prepareConfigFile returned error: %v", err)
	}

	if cfg.Container != "test_container" {
		t.Errorf("Expected Container to be 'test_container', got '%s'", cfg.Container)
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2023-10-04T12:00:00Z")
	if !cfg.Created.Equal(expectedTime) {
		t.Errorf("Expected Created time to be '%v', got '%v'", expectedTime, cfg.Created)
	}

	title := cfg.Config.Labels["org.opencontainers.image.title"]
	if title != "test_title" {
		t.Errorf("Expected label 'org.opencontainers.image.title' to be 'test_title', got '%s'", title)
	}
}

// TestPrepareLayers tests prepareLayers by creating test files and verifying the layers.
func TestPrepareLayers(t *testing.T) {
	t.Run("WithoutOmitLayersContent", func(t *testing.T) {
		dir := t.TempDir()
		filenames := []string{"file1.txt", "file2.txt"}

		// Create test files
		for _, name := range filenames {
			filePath := filepath.Join(dir, name)
			err := os.WriteFile(filePath, []byte("test content"), 0644)
			require.NoError(t, err, "Failed to write test file '%s'", name)
		}

		// Prepare options without omitting layer content
		opts := makeOptions()
		opts.omitLayersContent = false

		// Create a dummy config file (not required in this case but included for completeness)
		cfgFile := &v1.ConfigFile{}

		layers, err := prepareLayers(dir, cfgFile, opts)
		require.NoError(t, err, "prepareLayers returned error")

		// Since files are split into chunks, we need to calculate the expected number of layers
		expectedLayerCount := 0
		for _, name := range filenames {
			filePath := filepath.Join(dir, name)
			info, err := os.Stat(filePath)
			require.NoError(t, err, "Failed to stat file '%s'", name)
			chunks := int((info.Size() + int64(opts.chunkSize) - 1) / int64(opts.chunkSize))
			expectedLayerCount += chunks
		}

		assert.Equal(t, expectedLayerCount, len(layers), "Expected %d layers, got %d", expectedLayerCount, len(layers))
	})

	t.Run("WithOmitLayersContent", func(t *testing.T) {
		dir := t.TempDir()

		// Prepare options with omitting layer content
		opts := makeOptions()
		opts.omitLayersContent = true

		// Create a config file with RootFS.DiffIDs
		cfgFile := &v1.ConfigFile{
			RootFS: v1.RootFS{
				Type: "layers",
				DiffIDs: []v1.Hash{
					{Algorithm: "sha256", Hex: "1111111111111111111111111111111111111111111111111111111111111111"},
					{Algorithm: "sha256", Hex: "2222222222222222222222222222222222222222222222222222222222222222"},
				},
			},
		}

		// Write the config file to dir
		cfgData, err := json.Marshal(cfgFile)
		require.NoError(t, err, "Failed to marshal config file")
		err = os.WriteFile(filepath.Join(dir, LocalConfigFilename), cfgData, 0644)
		require.NoError(t, err, "Failed to write config file")

		// Create a manifest file using v1.Manifest
		manifest := &v1.Manifest{
			SchemaVersion: 2,
			MediaType:     types.OCIManifestSchema1,
			Config: v1.Descriptor{
				MediaType: types.OCIConfigJSON,
				Size:      int64(len(cfgData)),
				Digest:    v1.Hash{Algorithm: "sha256", Hex: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			},
			Layers: []v1.Descriptor{
				{
					MediaType:   types.MediaType("application/online.jarosik.tomasz.geranos.segment"),
					Size:        12345,
					Digest:      v1.Hash{Algorithm: "sha256", Hex: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
					Annotations: map[string]string{"filename": "file1.txt", "range": "0-12345"},
				},
				{
					MediaType:   types.MediaType("application/online.jarosik.tomasz.geranos.segment"),
					Size:        67890,
					Digest:      v1.Hash{Algorithm: "sha256", Hex: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
					Annotations: map[string]string{"filename": "file2.txt", "range": "12346-80235"},
				},
			},
		}

		// Write the manifest file to dir
		manifestData, err := json.Marshal(manifest)
		require.NoError(t, err, "Failed to marshal manifest file")
		err = os.WriteFile(filepath.Join(dir, LocalManifestFilename), manifestData, 0644)
		require.NoError(t, err, "Failed to write manifest file")

		// Read the config file back (to simulate real usage)
		cfgFileRead, err := prepareConfigFile(dir, true)
		require.NoError(t, err, "prepareConfigFile returned error")

		layers, err := prepareLayers(dir, cfgFileRead, opts)
		require.NoError(t, err, "prepareLayers returned error")

		expectedLayerCount := len(cfgFile.RootFS.DiffIDs)
		assert.Equal(t, expectedLayerCount, len(layers), "Expected %d layers, got %d", expectedLayerCount, len(layers))

		// Verify that layers are placeholder layers without content and have correct annotations
		for i, l := range layers {
			// Attempting to access content should result in an error
			_, err := l.Compressed()
			assert.Error(t, err, "Expected error when accessing compressed content of placeholder layer")
			_, err = l.Uncompressed()
			assert.Error(t, err, "Expected error when accessing uncompressed content of placeholder layer")

			// Verify that digest and diffID match the precomputed values
			digest, err := l.Digest()
			require.NoError(t, err, "Failed to get digest")
			diffID, err := l.DiffID()
			require.NoError(t, err, "Failed to get diffID")

			// Compare with expected values from the manifest and config
			expectedDigest := manifest.Layers[i].Digest
			expectedDiffID := cfgFile.RootFS.DiffIDs[i]

			assert.Equal(t, expectedDigest.String(), digest.String(), "Digest mismatch at layer %d", i)
			assert.Equal(t, expectedDiffID.String(), diffID.String(), "DiffID mismatch at layer %d", i)

			// Verify annotations
			if la, ok := l.(interface{ Annotations() map[string]string }); ok {
				annotations := la.Annotations()
				expectedAnnotations := manifest.Layers[i].Annotations
				assert.Equal(t, expectedAnnotations, annotations, "Annotations mismatch at layer %d", i)
			} else {
				t.Errorf("Layer at index %d does not implement Annotations()", i)
			}
		}
	})
}

func TestPrepareAddendums(t *testing.T) {
	// Create dummy layers using filesegment.NewLayer
	dir := t.TempDir()
	filePath1 := filepath.Join(dir, "layer1.txt")
	filePath2 := filepath.Join(dir, "layer2.txt")

	// Write test content to the files
	err := os.WriteFile(filePath1, []byte("content1"), 0644)
	assert.NoError(t, err, "Failed to write to file %s", filePath1)
	err = os.WriteFile(filePath2, []byte("content2"), 0644)
	assert.NoError(t, err, "Failed to write to file %s", filePath2)

	// Create layers using filesegment.NewLayer
	layer1, err := filesegment.NewLayer(filePath1)
	assert.NoError(t, err, "Failed to create layer1")
	layer2, err := filesegment.NewLayer(filePath2)
	assert.NoError(t, err, "Failed to create layer2")

	// Convert layers to []v1.Layer
	layers := []v1.Layer{layer1, layer2}

	// Call prepareAddendums
	addendums, err := prepareAddendums(layers)
	assert.NoError(t, err, "prepareAddendums returned an error")

	// Verify the number of addendums matches the number of layers
	assert.Equal(t, len(layers), len(addendums), "Number of addendums should match number of layers")

	// Verify that each addendum corresponds to the correct layer
	for i, addendum := range addendums {
		assert.Equal(t, layers[i], addendum.Layer, "Addendum layer does not match expected layer at index %d", i)
		// Optionally, verify media type and annotations
		mt, err := layers[i].MediaType()
		assert.NoError(t, err, "Failed to get media type for layer at index %d", i)
		assert.Equal(t, mt, addendum.MediaType, "Media type mismatch at index %d", i)
	}
}

// TestPrepareDiffIDs tests prepareDiffIDs by checking that DiffIDs are prepared correctly.
func TestPrepareDiffIDs(t *testing.T) {
	// Create dummy layers using filesegment.NewLayer
	dir := t.TempDir()
	filePath1 := filepath.Join(dir, "layer1.txt")
	filePath2 := filepath.Join(dir, "layer2.txt")

	// Write test content to the files
	err := os.WriteFile(filePath1, []byte("content1"), 0644)
	require.NoError(t, err, "Failed to write to file %s", filePath1)
	err = os.WriteFile(filePath2, []byte("content2"), 0644)
	require.NoError(t, err, "Failed to write to file %s", filePath2)

	// Create layers using filesegment.NewLayer
	layer1, err := filesegment.NewLayer(filePath1)
	require.NoError(t, err, "Failed to create layer1")
	layer2, err := filesegment.NewLayer(filePath2)
	require.NoError(t, err, "Failed to create layer2")

	// Convert layers to []v1.Layer
	layers := []v1.Layer{layer1, layer2}

	// Call prepareDiffIDs
	diffIDs, err := prepareDiffIDs(layers)
	require.NoError(t, err, "prepareDiffIDs returned an error")

	// Verify the number of diffIDs matches the number of layers
	require.Equal(t, len(layers), len(diffIDs), "Number of diffIDs should match number of layers")

	// Verify that each diffID is valid
	for i, diffID := range diffIDs {
		assert.Equal(t, "sha256", diffID.Algorithm, "Expected diffID algorithm to be 'sha256' for layer %d", i)
		assert.NotEmpty(t, diffID.Hex, "Expected diffID hex to be non-empty for layer %d", i)
	}
}

// TestPrepareImage tests prepareImage by creating an image with dummy config and addendums.
func TestPrepareImage(t *testing.T) {
	cfg := &v1.ConfigFile{
		Container: "test_container",
		Created:   v1.Time{Time: time.Now()},
		Config: v1.Config{
			Labels: map[string]string{
				"test_label": "test_value",
			},
		},
	}

	// Create dummy layers using filesegment.NewLayer
	dir := t.TempDir()
	filePath := filepath.Join(dir, "layer.txt")
	os.WriteFile(filePath, []byte("content"), 0644)

	layer, err := filesegment.NewLayer(filePath)
	if err != nil {
		t.Fatalf("Failed to create layer: %v", err)
	}

	addendums := []mutate.Addendum{
		{
			Layer:       layer,
			History:     v1.History{},
			Annotations: layer.Annotations(),
			MediaType:   filesegment.MediaType,
		},
	}

	img, err := prepareImage(cfg, addendums)
	if err != nil {
		t.Fatalf("prepareImage returned error: %v", err)
	}

	imageCfg, err := img.ConfigFile()
	if err != nil {
		t.Fatalf("Failed to get image config: %v", err)
	}

	if imageCfg.Container != "test_container" {
		t.Errorf("Expected Container to be 'test_container', got '%s'", imageCfg.Container)
	}

	if imageCfg.Config.Labels["test_label"] != "test_value" {
		t.Errorf("Expected label 'test_label' to be 'test_value', got '%s'", imageCfg.Config.Labels["test_label"])
	}
}

// TestComputeRootFS tests computeRootFS by verifying root filesystem and bytes read.
func TestComputeRootFS(t *testing.T) {
	// Set up test files and content
	dir := t.TempDir()
	filePath1 := filepath.Join(dir, "layer1.txt")
	filePath2 := filepath.Join(dir, "layer2.txt")
	content1 := []byte("content1")
	content2 := []byte("content2")

	// Write test content to files
	err := os.WriteFile(filePath1, content1, 0644)
	require.NoError(t, err, "Failed to write to file %s", filePath1)
	err = os.WriteFile(filePath2, content2, 0644)
	require.NoError(t, err, "Failed to write to file %s", filePath2)

	// Create layers using filesegment.NewLayer
	layer1, err := filesegment.NewLayer(filePath1)
	require.NoError(t, err, "Failed to create layer1")
	layer2, err := filesegment.NewLayer(filePath2)
	require.NoError(t, err, "Failed to create layer2")

	// Convert layers to []v1.Layer
	layers := []v1.Layer{layer1, layer2}

	// Set up context and options
	ctx := context.Background()
	opts := makeOptions()

	// Call computeRootFS
	rootFS, bytesReadCount, err := computeRootFS(ctx, layers, opts)
	require.NoError(t, err, "computeRootFS returned an error")

	// Expected bytes read: sum of content lengths * 2 (for DiffID and Digest)
	// Note: Since the actual bytes read may vary due to compression, we check for minimum expected bytes
	minExpectedBytesRead := int64(len(content1)+len(content2)) * 2
	assert.GreaterOrEqual(t, bytesReadCount, minExpectedBytesRead, "bytesReadCount should be at least %d", minExpectedBytesRead)

	// Verify that rootFS contains the correct number of DiffIDs
	assert.Equal(t, len(layers), len(rootFS.DiffIDs), "Number of DiffIDs should match number of layers")

	// Verify that each DiffID is valid
	for i, diffID := range rootFS.DiffIDs {
		assert.Equal(t, "sha256", diffID.Algorithm, "Expected DiffID algorithm to be 'sha256' for layer %d", i)
		assert.NotEmpty(t, diffID.Hex, "Expected DiffID hex to be non-empty for layer %d", i)
	}
}

// TestRead tests the Read function with various scenarios using subtests.
func TestRead(t *testing.T) {
	t.Run("TestReadSuccess", func(t *testing.T) {
		// Setup: Create a temporary directory with test files
		dir := t.TempDir()
		filenames := []string{"file1.txt", "file2.txt"}
		for _, name := range filenames {
			filePath := filepath.Join(dir, name)
			err := os.WriteFile(filePath, []byte("test content for "+name), 0644)
			require.NoError(t, err, "Failed to write test file '%s'", name)
		}

		// Call Read()
		ctx := context.Background()
		img, err := Read(ctx, dir)
		require.NoError(t, err, "Read returned an error")
		assert.NotNil(t, img, "DirImage should not be nil")

		// Verify that the image has layers
		manifest, err := img.Image.Manifest()
		require.NoError(t, err, "Failed to get manifest from image")
		assert.NotEmpty(t, manifest.Layers, "Image should have layers")
	})

	t.Run("TestReadWithOmitLayersContent", func(t *testing.T) {
		// Setup: Create a temporary directory with config and manifest files
		dir := t.TempDir()

		// Prepare options with omitting layer content
		opts := []Option{
			WithOmitLayersContent(),
		}

		// Create a config file with RootFS.DiffIDs
		cfgFile := &v1.ConfigFile{
			RootFS: v1.RootFS{
				Type: "layers",
				DiffIDs: []v1.Hash{
					{Algorithm: "sha256", Hex: "1111111111111111111111111111111111111111111111111111111111111111"},
					{Algorithm: "sha256", Hex: "2222222222222222222222222222222222222222222222222222222222222222"},
				},
			},
			Created: v1.Time{Time: time.Now()},
		}

		// Write the config file to dir
		cfgData, err := json.Marshal(cfgFile)
		require.NoError(t, err, "Failed to marshal config file")
		err = os.WriteFile(filepath.Join(dir, LocalConfigFilename), cfgData, 0644)
		require.NoError(t, err, "Failed to write config file")

		// Create a manifest file using v1.Manifest
		manifest := &v1.Manifest{
			SchemaVersion: 2,
			MediaType:     types.OCIManifestSchema1,
			Config: v1.Descriptor{
				MediaType: types.OCIConfigJSON,
				Size:      int64(len(cfgData)),
				Digest:    v1.Hash{Algorithm: "sha256", Hex: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			},
			Layers: []v1.Descriptor{
				{
					MediaType:   types.MediaType("application/online.jarosik.tomasz.geranos.segment"),
					Size:        12345,
					Digest:      v1.Hash{Algorithm: "sha256", Hex: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
					Annotations: map[string]string{"filename": "file1.txt", "range": "0-12345"},
				},
				{
					MediaType:   types.MediaType("application/online.jarosik.tomasz.geranos.segment"),
					Size:        67890,
					Digest:      v1.Hash{Algorithm: "sha256", Hex: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
					Annotations: map[string]string{"filename": "file2.txt", "range": "12346-80235"},
				},
			},
		}

		// Write the manifest file to dir
		manifestData, err := json.Marshal(manifest)
		require.NoError(t, err, "Failed to marshal manifest file")
		err = os.WriteFile(filepath.Join(dir, LocalManifestFilename), manifestData, 0644)
		require.NoError(t, err, "Failed to write manifest file")

		// Call Read()
		ctx := context.Background()
		img, err := Read(ctx, dir, opts...)
		require.NoError(t, err, "Read returned an error")
		assert.NotNil(t, img, "DirImage should not be nil")

		// Verify that the image has the expected number of layers
		manifestFromImage, err := img.Image.Manifest()
		require.NoError(t, err, "Failed to get manifest from image")
		assert.Equal(t, len(manifest.Layers), len(manifestFromImage.Layers), "Number of layers should match")

		assert.Equal(t, int64(0), img.BytesReadCount.Load())
	})

	t.Run("TestReadInvalidDirectory", func(t *testing.T) {
		// Call Read() with an invalid directory
		ctx := context.Background()
		_, err := Read(ctx, "/non/existent/directory")
		require.Error(t, err, "Expected error when reading from invalid directory")
	})

	t.Run("TestReadWithMissingConfig", func(t *testing.T) {
		// Setup: Create a temporary directory without config file
		dir := t.TempDir()

		// Set options that require config file
		opts := []Option{
			WithOmitLayersContent(),
		}

		// Call Read()
		ctx := context.Background()
		_, err := Read(ctx, dir, opts...)
		require.Error(t, err, "Expected error when config file is missing")
	})

	t.Run("TestReadWithInvalidConfig", func(t *testing.T) {
		// Setup: Create a temporary directory with an invalid config file
		dir := t.TempDir()

		// Write an invalid config.json file
		err := os.WriteFile(filepath.Join(dir, LocalConfigFilename), []byte("invalid json"), 0644)
		require.NoError(t, err, "Failed to write invalid config file")

		// Call Read()
		ctx := context.Background()
		_, err = Read(ctx, dir)
		require.Error(t, err, "Expected error when config file is invalid")
	})

	t.Run("TestReadWithNoFiles", func(t *testing.T) {
		// Setup: Create a temporary directory with no files
		dir := t.TempDir()

		// Call Read()
		ctx := context.Background()
		img, err := Read(ctx, dir)
		require.NoError(t, err, "Read returned an error")
		assert.NotNil(t, img, "DirImage should not be nil")

		// Verify that the image has no layers
		manifest, err := img.Image.Manifest()
		require.NoError(t, err, "Failed to get manifest from image")
		assert.Empty(t, manifest.Layers, "Expected no layers in the image")
	})

	t.Run("ChunkSizeOption", func(t *testing.T) {
		dir := t.TempDir()
		fileName := "largefile.txt"
		filePath := filepath.Join(dir, fileName)
		// Create a file with 10 bytes
		content := []byte("1234567890")
		err := os.WriteFile(filePath, content, 0644)
		require.NoError(t, err, "Failed to write test file")

		// Test with chunkSize = 3, expecting 4 layers (chunks of 3, 3, 3, 1 bytes)
		ctx := context.Background()
		chunkSize := int64(3)
		img, err := Read(ctx, dir, WithChunkSize(chunkSize))
		require.NoError(t, err, "Read returned error")
		require.NotNil(t, img, "Expected img to be non-nil")

		layers, err := img.Image.Layers()
		require.NoError(t, err, "Failed to get image layers")
		assert.Equal(t, 4, len(layers), "Expected four layers due to chunking")

		// Test with chunkSize = 5, expecting 2 layers (chunks of 5, 5 bytes)
		chunkSize = 5
		img, err = Read(ctx, dir, WithChunkSize(chunkSize))
		require.NoError(t, err, "Read returned error")
		require.NotNil(t, img, "Expected img to be non-nil")

		layers, err = img.Image.Layers()
		require.NoError(t, err, "Failed to get image layers")
		assert.Equal(t, 2, len(layers), "Expected two layers due to chunking")
	})

	t.Run("MultipleFilesWithChunking", func(t *testing.T) {
		dir := t.TempDir()
		fileNames := []string{"file1.txt", "file2.txt"}
		contents := []string{"1234567890", "abcdefghij"} // Both files are 10 bytes
		for i, fileName := range fileNames {
			filePath := filepath.Join(dir, fileName)
			content := []byte(contents[i])
			err := os.WriteFile(filePath, content, 0644)
			require.NoError(t, err, "Failed to write test file: %s", fileName)
		}

		ctx := context.Background()
		chunkSize := int64(4)
		img, err := Read(ctx, dir, WithChunkSize(chunkSize))
		require.NoError(t, err, "Read returned error")
		require.NotNil(t, img, "Expected img to be non-nil")

		layers, err := img.Image.Layers()
		require.NoError(t, err, "Failed to get image layers")
		// Each 10-byte file will be split into 3 layers (chunks of 4, 4, 2 bytes)
		// Total layers expected: 2 files * 3 layers = 6 layers
		expectedLayers := 6
		assert.Equal(t, expectedLayers, len(layers), "Expected %d layers due to chunking", expectedLayers)
	})
}

// TestReadWriteReadOmitLayers tests reading an image, writing it, reading it back with omitLayersContent,
// and comparing the manifests and digests to ensure they are the same.
func TestReadWriteReadOmitLayers(t *testing.T) {
	ctx := context.Background()

	// Step 1: Read image from a directory
	sourceDir := t.TempDir()

	// Create some test files in sourceDir
	fileNames := []string{"file1.txt", "file2.txt"}
	contents := []string{"This is file1 content", "This is file2 extra content"}
	for i, name := range fileNames {
		err := os.WriteFile(filepath.Join(sourceDir, name), []byte(contents[i]), 0644)
		require.NoError(t, err, "Failed to write test file %s", name)
	}

	// Read the directory into an image
	img1, err := Read(ctx, sourceDir)
	require.NoError(t, err, "Failed to read image from source directory")
	require.NotNil(t, img1, "img1 should not be nil")

	assert.Equal(t, int64(96), img1.BytesReadCount.Load())

	// Get the manifest and digest from img1
	_, err = img1.Image.Manifest()
	require.NoError(t, err, "Failed to get manifest from img1")
	digest1, err := img1.Image.Digest()
	require.NoError(t, err, "Failed to get digest from img1")

	// Step 2: Write the image to another directory
	destDir := t.TempDir()
	err = img1.Write(ctx, destDir)
	require.NoError(t, err, "Failed to write image to destination directory")

	// Step 3: Read the image back with omitLayersContent=true
	img2, err := Read(ctx, destDir, WithOmitLayersContent())
	require.NoError(t, err, "Failed to read image from destination directory with omitLayersContent=true")
	require.NotNil(t, img2, "img2 should not be nil")
	assert.Equal(t, int64(0), img2.BytesReadCount.Load())

	// Get the manifest and digest from img2
	//manifest2, err := img2.Image.Manifest()
	//require.NoError(t, err, "Failed to get manifest from img2")
	digest2, err := img2.Image.Digest()
	require.NoError(t, err, "Failed to get digest from img2")

	// Compare manifests by comparing their JSON representations
	manifest1JSON, err := img1.Image.RawManifest()
	require.NoError(t, err, "Failed to get raw manifest from img1")
	manifest2JSON, err := img2.Image.RawManifest()
	require.NoError(t, err, "Failed to get raw manifest from img2")
	assert.Equal(t, manifest1JSON, manifest2JSON, "Raw manifests should be equal")

	// Compare digests
	assert.Equal(t, digest1, digest2, "Digests should be equal")

	// Get the raw config JSON from both images
	configJSON1, err := img1.Image.RawConfigFile()
	require.NoError(t, err, "Failed to get raw config JSON from img1")
	configJSON2, err := img2.Image.RawConfigFile()
	require.NoError(t, err, "Failed to get raw config JSON from img2")

	// Compare the raw config JSON
	assert.Equal(t, configJSON1, configJSON2, "Raw config JSON should be equal")

	configFile1, err := img1.Image.ConfigFile()
	require.NoError(t, err, "Failed to get config file from img1")
	configFile2, err := img2.Image.ConfigFile()
	require.NoError(t, err, "Failed to get config file from img2")
	// Compare the Created times using Equal()
	assert.True(t, configFile1.Created.Time.Equal(configFile2.Created.Time), "Created times should be equal")

	// Optionally, compare the UnixNano timestamps
	assert.Equal(t, configFile1.Created.Time.UnixNano(), configFile2.Created.Time.UnixNano(), "Created times' UnixNano values should be equal")

	// In Go, time.Time can include a monotonic clock reading, used for measuring elapsed time with high precision.
	// This monotonic clock reading is not part of the wall-clock time and is not included when serializing time.Time to JSON or other formats.
	// Ignore it here
	configFile2.Created = configFile1.Created
	assert.Equal(t, configFile1, configFile2, "Config files should be equal")
}
