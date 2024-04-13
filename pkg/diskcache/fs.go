package diskcache

import (
	"errors"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/tomekjarosik/geranos/pkg/filesegment"
	"github.com/tomekjarosik/geranos/pkg/sparsefile"
	"io"
	"os"
	"path/filepath"
)

type fscache struct {
	path string
	mt   types.MediaType
}

func NewFilesystemCache(path string) cache.Cache {
	return &fscache{path, ""}
}

type teeLayer struct {
	v1.Layer

	path           string
	digest, diffID v1.Hash
}

func (l *teeLayer) create(digest v1.Hash, diffID v1.Hash) (io.WriteCloser, error) {
	if err := os.MkdirAll(l.path, 0700); err != nil {
		return nil, fmt.Errorf("unable to create directories: %w", err)
	}
	f, err := os.Create(cachepath(l.path, digest))
	if err != nil {
		return nil, fmt.Errorf("unable to create file: %w", err)
	}
	err = os.WriteFile(cachepath(l.path, diffID)+".link", []byte(digest.String()), 0644)
	if err != nil {
		return nil, err
	}
	return sparsefile.NewWriter(f), nil
}

func (l *teeLayer) Compressed() (io.ReadCloser, error) {
	return nil, errors.New("unsupported")
}

func (l *teeLayer) Uncompressed() (io.ReadCloser, error) {
	f, err := l.create(l.digest, l.diffID)
	if err != nil {
		return nil, fmt.Errorf("unable to create cached layer: %w", err)
	}
	rc, err := l.Layer.Uncompressed()
	if err != nil {
		return nil, fmt.Errorf("unable to get uncompressed layer: %w", err)
	}
	return &readcloser{
		t:      io.TeeReader(rc, f),
		closes: []func() error{rc.Close, f.Close},
	}, nil
}

type readcloser struct {
	t      io.Reader
	closes []func() error
}

func (rc *readcloser) Read(b []byte) (int, error) {
	return rc.t.Read(b)
}

func (rc *readcloser) Close() error {
	var err error
	for _, c := range rc.closes {
		lastError := c()
		if err == nil {
			err = lastError
		}
	}
	return err
}

func (fs *fscache) Put(l v1.Layer) (v1.Layer, error) {
	digest, err := l.Digest()
	if err != nil {
		return nil, err
	}
	diffID, err := l.DiffID()
	if err != nil {
		return nil, err
	}
	return &teeLayer{
		Layer:  l,
		path:   fs.path,
		digest: digest,
		diffID: diffID,
	}, nil
}

func (fs *fscache) Get(h v1.Hash) (v1.Layer, error) {
	// hash could be Digest (recommended) or DiffId (fallback)
	shaFilename := cachepath(fs.path, h)
	f, err := os.Open(shaFilename)
	defer f.Close()
	isDigest := true
	if os.IsNotExist(err) {
		// Is not a "Digest" fallback to uncompressed layer: .link
		linkContent, err2 := os.ReadFile(shaFilename + ".link")
		if os.IsNotExist(err2) {
			return nil, cache.ErrNotFound
		}
		if err2 != nil {
			return nil, err2
		}
		shaFilename = filepath.Join(fs.path, string(linkContent))
		isDigest = false
		err = nil
	}
	l, err := filesegment.NewLayer(shaFilename)
	var hashCalculated v1.Hash
	if err == nil {
		if isDigest {
			hashCalculated, _ = l.Digest()
		} else {
			hashCalculated, _ = l.DiffID()
		}
	}
	// Below code handle cases of when cache is corrupted
	if hashCalculated != h {
		fmt.Printf("isDigest=%v, expected hash %v, got corrupted hash %v, shaFilename: %v\n", isDigest, h, hashCalculated, shaFilename)
		if err := fs.Delete(h); err != nil {
			return nil, err
		}
		return nil, cache.ErrNotFound
	}
	return l, nil
}

func (fs *fscache) Delete(h v1.Hash) error {
	err := os.RemoveAll(cachepath(fs.path, h))
	if os.IsNotExist(err) {
		return os.ErrNotExist
	}
	return err
}

func cachepath(path string, h v1.Hash) string {
	return filepath.Join(path, h.String())
}
