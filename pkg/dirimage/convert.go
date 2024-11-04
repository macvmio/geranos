package dirimage

import (
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/macvmio/geranos/pkg/filesegment"
	"sync/atomic"
)

func Convert(img v1.Image) (*DirImage, error) {
	manifest, err := img.Manifest()
	if err != nil {
		return nil, err
	}

	// Retrieve the config file to access root filesystem diffIDs
	configFile, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}
	diffIDs := configFile.RootFS.DiffIDs

	// Ensure the number of diffIDs matches the number of layers
	if len(diffIDs) != len(manifest.Layers) {
		return nil, fmt.Errorf("mismatch between diffIDs (%d) and manifest layers (%d)", len(diffIDs), len(manifest.Layers))
	}

	segmentDescriptors := make([]*filesegment.Descriptor, 0)
	for i, l := range manifest.Layers {
		d, err := filesegment.ParseDescriptor(l, diffIDs[i])
		if err != nil {
			return nil, err
		}
		segmentDescriptors = append(segmentDescriptors, d)
	}
	return &DirImage{
		Image:              img,
		BytesReadCount:     atomic.Int64{},
		directory:          "",
		segmentDescriptors: segmentDescriptors,
	}, nil
}
