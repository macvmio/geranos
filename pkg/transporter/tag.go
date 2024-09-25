package transporter

import (
	"fmt"
	"github.com/google/go-containerregistry/pkg/logs"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"log"
	"os"
)

func RetagRemotely(oldImageRef, newImageRef string, opt ...Option) error {
	logs.Progress = log.New(os.Stdout, "", log.LstdFlags)
	opts := makeOptions(opt...)

	// Parse the old image reference (e.g., my-image:1.0)
	oldRef, err := name.ParseReference(oldImageRef)
	if err != nil {
		return fmt.Errorf("unable to parse old reference '%v': %w", oldImageRef, err)
	}

	// Retrieve the image manifest and config from the registry
	img, err := remote.Image(oldRef, opts.remoteOptions...)
	if err != nil {
		return fmt.Errorf("unable to fetch image from registry: %w", err)
	}

	// Parse the new image reference (e.g., my-image:latest)
	newRef, err := name.ParseReference(newImageRef)
	if err != nil {
		return fmt.Errorf("unable to parse new reference '%v': %w", newImageRef, err)
	}

	// Push the image with the new reference (re-tagging it)
	if err := remote.Write(newRef, img, opts.remoteOptions...); err != nil {
		return fmt.Errorf("unable to push image with new tag: %w", err)
	}

	return nil
}
