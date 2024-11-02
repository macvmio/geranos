package filesegment

import (
	"errors"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/macvmio/geranos/pkg/zstd"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

const MediaType = types.MediaType("application/online.jarosik.tomasz.geranos.segment")

type Layer struct {
	filePath  string
	start     int64
	stop      int64
	mediaType types.MediaType
	diffID    v1.Hash

	hash             v1.Hash
	size             int64
	hashSizeError    error
	compressedOnce   sync.Once
	uncompressedOnce sync.Once

	log func(fmt string, args ...any)
}

var _ v1.Layer = (*Layer)(nil)

func (pfl *Layer) DiffID() (v1.Hash, error) {
	pfl.uncompressedOnce.Do(func() {
		rc, err := pfl.Uncompressed()
		if err != nil {
			return
		}
		defer rc.Close()
		cfgHash, _, err := v1.SHA256(rc)
		if err != nil {
			return
		}
		pfl.log("%v: calculated uncompressed layer hash", pfl)
		pfl.diffID = cfgHash
	})
	return pfl.diffID, nil
}

// Uncompressed implements v1.Layer
func (pfl *Layer) Uncompressed() (io.ReadCloser, error) {
	return newPartialFileReader(pfl.filePath, pfl.start, pfl.stop)
}

// Compressed implements v1.Layer
func (pfl *Layer) Compressed() (io.ReadCloser, error) {
	u, err := pfl.Uncompressed()
	if err != nil {
		return nil, err
	}
	return zstd.ReadCloser(u), nil
}

// Digest implements v1.Layer
func (pfl *Layer) Digest() (v1.Hash, error) {
	pfl.calcSizeHash()
	return pfl.hash, pfl.hashSizeError
}

func (pfl *Layer) calcSizeHash() {
	pfl.compressedOnce.Do(func() {
		var r io.ReadCloser
		r, pfl.hashSizeError = pfl.Compressed()
		if pfl.hashSizeError != nil {
			return
		}
		defer r.Close()
		pfl.hash, pfl.size, pfl.hashSizeError = v1.SHA256(r)
		pfl.log("%v: calculated compressed layer hash", pfl)
	})
}

func (pfl *Layer) MediaType() (types.MediaType, error) {
	return pfl.mediaType, nil
}

func (pfl *Layer) Size() (int64, error) {
	pfl.calcSizeHash()
	return pfl.size, pfl.hashSizeError
}

func (pfl *Layer) String() string {
	return fmt.Sprintf("layer from '%v' range[%v-%v]", filepath.Base(pfl.filePath), pfl.start, pfl.stop)
}

func (pfl *Layer) Start() int64 {
	return pfl.start
}

func (pfl *Layer) Stop() int64 {
	return pfl.stop
}

func (pfl *Layer) Annotations() map[string]string {
	return map[string]string{
		FilenameAnnotationKey: filepath.Base(pfl.filePath),
		RangeAnnotationKey:    fmt.Sprintf("%d-%d", pfl.start, pfl.stop),
	}
}

func (pfl *Layer) Length() int64 {
	return pfl.stop - pfl.start + 1
}

func NewLayer(filePath string, opts ...LayerOpt) (*Layer, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	pfl := &Layer{
		filePath:  filePath,
		start:     0,
		stop:      info.Size() - 1,
		mediaType: MediaType,
		log:       log.Printf,
	}
	for _, o := range opts {
		o(pfl)
	}
	if pfl.stop >= info.Size() {
		return nil, errors.New("provided 'stop' is outside of file size")
	}
	if pfl.start < 0 || pfl.start > pfl.stop {
		return nil, errors.New("provided 'start' index is out of range")
	}
	return pfl, nil
}
