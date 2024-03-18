package image

import (
	"errors"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func FromLayout(ref name.Reference, opts ...Option) (v1.Image, error) {
	return nil, errors.New("not implemented")
}

func ToLayout(img v1.Image, ref name.Reference, opts ...Option) error {

	return nil
}
