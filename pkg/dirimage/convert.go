package dirimage

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/mobileinf/geranos/pkg/filesegment"
)

func Convert(img v1.Image) (*DirImage, error) {
	manifest, err := img.Manifest()
	if err != nil {
		return nil, err
	}

	segmentDescriptors := make([]*filesegment.Descriptor, 0)
	for _, l := range manifest.Layers {
		d, err := filesegment.ParseDescriptor(l)
		if err != nil {
			return nil, err
		}
		segmentDescriptors = append(segmentDescriptors, d)
	}
	return &DirImage{
		Image:              img,
		BytesReadCount:     0,
		directory:          "",
		segmentDescriptors: segmentDescriptors,
	}, nil
}
