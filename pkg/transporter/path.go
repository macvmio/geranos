package transporter

import (
	"github.com/google/go-containerregistry/pkg/name"
	"os/user"
	"path/filepath"
	"strings"
)

func expandUser(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		u, err := user.Current()
		if err != nil {
			return path, err
		}
		path = filepath.Join(u.HomeDir, path[2:])
	}
	return path, nil
}

func mustExpandUser(path string) string {
	r, err := expandUser(path)
	if err != nil {
		panic(err)
	}
	return r
}

func PathFromRawRef(rawRef string, opt ...Option) (string, error) {
	opts := makeOptions(opt...)
	ref, err := name.ParseReference(rawRef, opts.refValidation)
	if err != nil {
		return "", err
	}
	return filepath.Join(opts.imagesPath, ref.Name()), nil
}
