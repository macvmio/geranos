package dirimage

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/mobileinf/geranos/pkg/filesegment"
)

type DirImage struct {
	v1.Image
	BytesReadCount    int64
	BytesWrittenCount int64
	BytesSkippedCount int64

	directory          string
	segmentDescriptors []*filesegment.Descriptor
}

var _ v1.Image = (*DirImage)(nil)

func (d *DirImage) Length() int64 {
	res := int64(0)
	for _, d := range d.segmentDescriptors {
		res += d.Length()
	}
	return res
}
