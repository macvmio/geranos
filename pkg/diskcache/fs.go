package diskcache

import (
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/cache"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/klauspost/compress/zstd"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
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

func (l *teeLayer) create(h v1.Hash) (io.WriteCloser, error) {
	if err := os.MkdirAll(l.path, 0700); err != nil {
		return nil, fmt.Errorf("unable to create directories: %w", err)
	}
	fmt.Printf("teeLayer::create %v\n", cachepath(l.path, h))
	f, err := os.Create(cachepath(l.path, h))
	if err != nil {
		return nil, fmt.Errorf("unable to create file: %w", err)
	}
	return zstd.NewWriter(f)
}

func (l *teeLayer) Compressed() (io.ReadCloser, error) {
	fmt.Printf("teeLayer::Compressed\n")
	f, err := l.create(l.digest)
	if err != nil {
		return nil, fmt.Errorf("unable to create cached layer: %w", err)
	}
	rc, err := l.Layer.Compressed()
	if err != nil {
		return nil, fmt.Errorf("unable to get compressed layer: %w", err)
	}
	return &readcloser{
		t:      io.TeeReader(rc, f),
		closes: []func() error{rc.Close, f.Close},
	}, nil
}

func (l *teeLayer) Uncompressed() (io.ReadCloser, error) {
	fmt.Printf("teeLayer::Uncompressed\n")
	f, err := l.create(l.digest)
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
	fmt.Printf("fscache::Put(digest=%s, diffId=%s\n", digest.String(), diffID.String())
	return &teeLayer{
		Layer:  l,
		path:   fs.path,
		digest: digest,
		diffID: diffID,
	}, nil
}

type alwaysCompressedLayer struct {
	filename       string
	compressedOnce sync.Once
	hash           v1.Hash
	size           int64
	hashSizeError  error
}

func (nl *alwaysCompressedLayer) Digest() (v1.Hash, error) {
	nl.calcSizeHash()
	return nl.hash, nl.hashSizeError
}

func (nl *alwaysCompressedLayer) calcSizeHash() {
	nl.compressedOnce.Do(func() {
		var r io.ReadCloser
		r, nl.hashSizeError = nl.Compressed()
		if nl.hashSizeError != nil {
			return
		}
		defer r.Close()
		nl.hash, nl.size, nl.hashSizeError = v1.SHA256(r)
		log.Printf("%v: calculated compressed layer hash", nl)
	})
}

func (nl *alwaysCompressedLayer) Compressed() (io.ReadCloser, error) {
	fmt.Printf("alwaysCompressedLayer::Compressed()")
	return os.Open(nl.filename)
}

func (nl *alwaysCompressedLayer) Size() (int64, error) {
	fi, err := os.Stat(nl.filename)
	if err != nil {
		return 0, err
	}
	return fi.Size(), err
}

func (nl *alwaysCompressedLayer) MediaType() (types.MediaType, error) {
	return "", nil // TODO:
}

func (fs *fscache) Get(h v1.Hash) (v1.Layer, error) {
	fmt.Printf("fscache::Get(%s)\n", h.String())
	_, err := os.Open(cachepath(fs.path, h))

	if os.IsNotExist(err) {
		fmt.Printf("cache.ErrNoFound: %v\n", err)
		return nil, cache.ErrNotFound
	}
	l, err := partial.CompressedToLayer(&alwaysCompressedLayer{filename: cachepath(fs.path, h)})
	if err != nil {
		return nil, err
	}
	hashCalculated, err := l.Digest()
	// Below code handle cases of when cache is corrupted
	if err != nil || hashCalculated != h {
		fmt.Printf("error in cache impl")
		if err := fs.Delete(h); err != nil {
			return nil, err
		}
		return nil, cache.ErrNotFound
	}
	return l, err
}

func (fs *fscache) Delete(h v1.Hash) error {
	err := os.RemoveAll(cachepath(fs.path, h))
	if os.IsNotExist(err) {
		return os.ErrNotExist
	}
	return err
}

func cachepath(path string, h v1.Hash) string {
	file := h.String()
	res := filepath.Join(path, file)
	fmt.Printf("cache path: %s\n", res)
	return res
}
