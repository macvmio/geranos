package dirimage

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/tomekjarosik/geranos/pkg/image/filesegment"
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
