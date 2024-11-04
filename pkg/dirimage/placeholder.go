package dirimage

import (
	"encoding/json"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/macvmio/geranos/pkg/filesegment"
	"io"
	"os"
	"path/filepath"
)

type placeholderLayer struct {
	digest      v1.Hash
	diffID      v1.Hash
	size        int64
	length      int64
	mediaType   types.MediaType
	annotations map[string]string
}

func (l *placeholderLayer) Digest() (v1.Hash, error) {
	return l.digest, nil
}

func (l *placeholderLayer) DiffID() (v1.Hash, error) {
	return l.diffID, nil
}

func (l *placeholderLayer) Size() (int64, error) {
	return l.size, nil
}

func (l *placeholderLayer) Length() int64 {
	return l.length
}

func (l *placeholderLayer) MediaType() (types.MediaType, error) {
	return l.mediaType, nil
}

func (l *placeholderLayer) Compressed() (io.ReadCloser, error) {
	return nil, fmt.Errorf("compressed content not available")
}

func (l *placeholderLayer) Uncompressed() (io.ReadCloser, error) {
	return nil, fmt.Errorf("uncompressed content not available")
}

func (l *placeholderLayer) UncompressedSize() (int64, error) {
	return l.size, nil
}

func (l *placeholderLayer) Annotations() map[string]string {
	return l.annotations
}

func readManifest(filePath string) (*v1.Manifest, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("unable to read manifest file: %w", err)
	}

	var manifest v1.Manifest
	err = json.Unmarshal(data, &manifest)
	if err != nil {
		return nil, fmt.Errorf("unable to parse manifest file: %w", err)
	}

	return &manifest, nil
}

func prepareLayersFromManifestAndConfig(dir string, cfgFile *v1.ConfigFile) ([]v1.Layer, error) {
	// Read the manifest file
	manifestPath := filepath.Join(dir, LocalManifestFilename)
	manifest, err := readManifest(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	// Ensure that the number of layers matches the number of diff IDs
	if len(manifest.Layers) != len(cfgFile.RootFS.DiffIDs) {
		return nil, fmt.Errorf("mismatch between number of layers in manifest and diff IDs in config")
	}

	// Construct placeholder layers
	layers := make([]v1.Layer, len(manifest.Layers))
	for i, mLayer := range manifest.Layers {
		d, err := filesegment.ParseDescriptor(mLayer, cfgFile.RootFS.DiffIDs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse descriptor: %w", err)
		}
		// Create a placeholder layer
		l := &placeholderLayer{
			mediaType:   d.MediaType(),
			digest:      d.Digest(),
			diffID:      cfgFile.RootFS.DiffIDs[i],
			size:        mLayer.Size,
			length:      0,
			annotations: d.Annotations(),
		}

		layers[i] = l
	}

	return layers, nil
}
