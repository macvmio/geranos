package transporter

import (
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/mobileinf/geranos/pkg/layout"
)

func ReadManifest(src string, opt ...Option) (*v1.Manifest, error) {
	opts := makeOptions(opt...)
	ref, err := name.ParseReference(src, name.StrictValidation)
	if err != nil {
		return nil, err
	}
	lm := layout.NewMapper(opts.imagesPath, opts.dirimageOptions...)
	return lm.ReadManifest(ref)
}

func ReadDigest(src string, opt ...Option) (v1.Hash, error) {
	opts := makeOptions(opt...)
	ref, err := name.ParseReference(src, name.StrictValidation)
	if err != nil {
		return v1.Hash{}, err
	}
	lm := layout.NewMapper(opts.imagesPath, opts.dirimageOptions...)
	return lm.ReadDigest(ref)
}
