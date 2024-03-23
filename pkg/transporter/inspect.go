package transporter

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/tomekjarosik/geranos/pkg/image"
)

func Inspect(rawRef string, opt ...Option) (string, error) {
	opts := makeOptions(opt...)
	ref, err := name.ParseReference(rawRef, name.StrictValidation)
	if err != nil {
		return "", fmt.Errorf("unable to parse reference: %w", err)
	}
	lm := image.NewLayoutMapper(opts.imagesPath)
	img, err := lm.Read(ref)
	if err != nil {
		return "", fmt.Errorf("unable to read from ref %v: %w", ref, err)
	}
	cfg, err := img.ConfigFile()
	if err != nil {
		return "", fmt.Errorf("unable to get config file: %w", err)
	}
	outCfg, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return "", fmt.Errorf("unable to marshal config to json: %w", err)
	}
	manifest, err := img.Manifest()
	if err != nil {
		return "", fmt.Errorf("unable to extract manifest from image: %w", err)
	}
	outManifest, err := json.MarshalIndent(manifest, "", "\t")
	if err != nil {
		return "", fmt.Errorf("unable to marshal manifest to json: %w", err)
	}
	return string(outCfg) + "\n" + string(outManifest), nil
}
