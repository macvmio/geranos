package image

import (
	"errors"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/tomekjarosik/geranos/pkg/image/segmentlayer"
	"strconv"
	"strings"
)

type fileSegmentRecipe struct {
	Filename string
	Start    int64
	Stop     int64
	Digest   v1.Hash
}

type fileRecipe struct {
	Filename string
	Segments []fileSegmentRecipe
}

func (fr *fileRecipe) Size() int64 {
	if len(fr.Segments) == 0 {
		return 0
	}
	return fr.Segments[len(fr.Segments)-1].Stop + 1
}

func (fr *fileRecipe) Validate() error {
	if len(fr.Segments) == 0 {
		return errors.New("0 segments")
	}
	if fr.Segments[0].Start != 0 {
		return errors.New("first segment does not start from 0")
	}
	last := fr.Segments[0].Stop
	for i := 1; i < len(fr.Segments); i++ {
		s := &fr.Segments[1]
		if s.Start != last+1 {
			return fmt.Errorf("segment #%d has invalid start position %d, expected %d", i, s.Start, last+1)
		}
		if s.Stop < s.Start {
			return fmt.Errorf("segment #%d has Stop value (%d) lower thatn Start value (%d)", i, s.Start, s.Stop)
		}
	}
	return nil
}

func (fsr *fileSegmentRecipe) Length() int64 {
	return fsr.Stop - fsr.Start + 1
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

func createFileRecipesFromImage(img v1.Image) ([]*fileRecipe, error) {
	manifest, err := img.Manifest()
	if err != nil {
		return nil, err
	}

	fileRecipesMap := make(map[string]*fileRecipe)
	for _, l := range manifest.Layers {
		if l.MediaType != segmentlayer.FileSegmentMediaType {
			return nil, errors.New("unsupported layer type")
		}
		filename, present := l.Annotations[FilenameAnnotationKey]
		if !present {
			return nil, errors.New("missing filename annotation")
		}
		rangeString, present := l.Annotations[RangeAnnotationKey]
		if !present {
			return nil, errors.New("missing range annotation")
		}
		start, stop, err := parseIntPair(rangeString)
		if err != nil {
			return nil, fmt.Errorf("invalid range: %w", err)
		}
		fr, present := fileRecipesMap[filename]
		if !present {
			fr = &fileRecipe{
				Filename: filename,
				Segments: make([]fileSegmentRecipe, 0),
			}
		}
		fr.Segments = append(fr.Segments, fileSegmentRecipe{
			Filename: filename,
			Start:    start,
			Stop:     stop,
			Digest:   l.Digest,
		})
		fileRecipesMap[filename] = fr
	}
	res := make([]*fileRecipe, 0)
	for _, v := range fileRecipesMap {
		err = v.Validate()
		if err != nil {
			return nil, fmt.Errorf("file recipe for '%s' failed with: %w", v.Filename, err)
		}
		res = append(res, v)
	}
	return res, nil
}
