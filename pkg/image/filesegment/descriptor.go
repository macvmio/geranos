package filesegment

import (
	"errors"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"strconv"
	"strings"
)

const FilenameAnnotationKey = "filename"
const RangeAnnotationKey = "range"

type Descriptor struct {
	filename string
	start    int64
	stop     int64
	digest   v1.Hash
}

func (d *Descriptor) Filename() string {
	return d.filename
}

func (d *Descriptor) Start() int64 {
	return d.start
}

func (d *Descriptor) Stop() int64 {
	return d.stop
}

func (d *Descriptor) Digest() v1.Hash {
	return d.digest
}

func (d *Descriptor) Length() int64 {
	return d.stop - d.start + 1
}

func (d *Descriptor) Annotations() map[string]string {
	return map[string]string{
		FilenameAnnotationKey: d.filename,
		RangeAnnotationKey:    fmt.Sprintf("%d-%d", d.start, d.stop),
	}
}

func (d *Descriptor) MediaType() types.MediaType {
	return MediaType
}

// parseIntPair parses a string formatted as "<int>-<int>" and returns the two int64 numbers or an error.
func parseIntPair(s string) (int64, int64, error) {
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return 0, 0, errors.New("incorrect format, expected '<int>-<int>'")
	}

	firstInt, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, 0, errors.New("failed to parse the first integer as int64")
	}

	secondInt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return 0, 0, errors.New("failed to parse the second integer as int64")
	}

	return firstInt, secondInt, nil
}

func NewDescriptor(filename string, start, stop int64, digest v1.Hash) *Descriptor {
	return &Descriptor{
		filename: filename,
		start:    start,
		stop:     stop,
		digest:   digest,
	}
}

func ParseDescriptor(d v1.Descriptor) (*Descriptor, error) {
	if d.MediaType != MediaType {
		return nil, errors.New("unsupported layer type")
	}
	filename, present := d.Annotations[FilenameAnnotationKey]
	if !present {
		return nil, errors.New("missing filename annotation")
	}
	rangeString, present := d.Annotations[RangeAnnotationKey]
	if !present {
		return nil, errors.New("missing range annotation")
	}
	start, stop, err := parseIntPair(rangeString)
	if err != nil {
		return nil, fmt.Errorf("invalid range: %w", err)
	}
	return &Descriptor{
		filename: filename,
		start:    start,
		stop:     stop,
		digest:   d.Digest,
	}, nil
}
