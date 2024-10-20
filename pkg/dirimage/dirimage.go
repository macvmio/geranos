package dirimage

import (
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/macvmio/geranos/pkg/filesegment"
	"sync/atomic"
)

// https://opencontainers.org/posts/blog/2024-03-13-image-and-distribution-1-1/
var ManifestMediaType = types.OCIManifestSchema1
var ConfigMediaType = types.OCIConfigJSON

const LocalManifestFilename = ".oci.manifest.json"
const LocalConfigFilename = ".oci.config.json"

type DirImage struct {
	v1.Image
	BytesReadCount    atomic.Int64
	BytesWrittenCount atomic.Int64
	BytesSkippedCount atomic.Int64

	directory          string
	segmentDescriptors []*filesegment.Descriptor
}

var _ v1.Image = (*DirImage)(nil)

func New(dir string, img v1.Image) *DirImage {
	return &DirImage{
		Image:     img,
		directory: dir,
	}
}

func (di *DirImage) Length() int64 {
	res := int64(0)
	for _, d := range di.segmentDescriptors {
		res += d.Length()
	}
	return res
}
