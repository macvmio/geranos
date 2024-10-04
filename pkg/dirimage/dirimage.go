package dirimage

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/mobileinf/geranos/pkg/filesegment"
)

// https://opencontainers.org/posts/blog/2024-03-13-image-and-distribution-1-1/
var ManifestMediaType = types.MediaType("application/vnd.oci.image.manifest.v1+json")
var ConfigMediaType = types.MediaType("application/vnd.oci.image.config.v1+json")

type DirImage struct {
	v1.Image
	BytesReadCount    int64
	BytesWrittenCount int64
	BytesSkippedCount int64

	directory          string
	segmentDescriptors []*filesegment.Descriptor
}

var _ v1.Image = (*DirImage)(nil)

func (di *DirImage) Length() int64 {
	res := int64(0)
	for _, d := range di.segmentDescriptors {
		res += d.Length()
	}
	return res
}
