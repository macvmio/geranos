package layout

import (
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

func NewMountableImage(image v1.Image, ref name.Reference) *MountableImage {
	return &MountableImage{
		Image:     image,
		Reference: ref,
	}
}

type MountableImage struct {
	v1.Image

	Reference name.Reference
}

func (mi *MountableImage) Layers() ([]v1.Layer, error) {
	ls, err := mi.Image.Layers()
	if err != nil {
		return nil, err
	}
	mls := make([]v1.Layer, 0, len(ls))
	for _, l := range ls {
		mls = append(mls, &remote.MountableLayer{
			Layer:     l,
			Reference: mi.Reference,
		})
	}
	return mls, nil
}

func (mi *MountableImage) LayerByDigest(d v1.Hash) (v1.Layer, error) {
	l, err := mi.Image.LayerByDigest(d)
	if err != nil {
		return nil, err
	}
	return &remote.MountableLayer{
		Layer:     l,
		Reference: mi.Reference,
	}, nil
}

func (mi *MountableImage) LayerByDiffID(d v1.Hash) (v1.Layer, error) {
	l, err := mi.Image.LayerByDiffID(d)
	if err != nil {
		return nil, err
	}
	return &remote.MountableLayer{
		Layer:     l,
		Reference: mi.Reference,
	}, nil
}

func (mi *MountableImage) ConfigLayer() (v1.Layer, error) {
	l, err := partial.ConfigLayer(mi.Image)
	if err != nil {
		return nil, err
	}
	return &remote.MountableLayer{
		Layer:     l,
		Reference: mi.Reference,
	}, nil
}
