package sketch

import (
	"errors"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/macvmio/geranos/pkg/filesegment"
)

type fileBlueprint struct {
	Filename string
	Segments []*filesegment.Descriptor
}

func (fr *fileBlueprint) Size() int64 {
	if len(fr.Segments) == 0 {
		return 0
	}
	return fr.Segments[len(fr.Segments)-1].Stop() + 1
}

func (fr *fileBlueprint) Validate() error {
	if len(fr.Segments) == 0 {
		return errors.New("0 segments")
	}
	if fr.Segments[0].Start() != 0 {
		return errors.New("first segment does not start from 0")
	}
	last := fr.Segments[0].Stop()
	for i := 1; i < len(fr.Segments); i++ {
		s := fr.Segments[i]
		if s.Start() != last+1 {
			return fmt.Errorf("segment #%d has invalid start position %d, expected %d", i, s.Start(), last+1)
		}
		if s.Stop() < s.Start() {
			return fmt.Errorf("segment #%d has Stop value (%d) lower thatn Start value (%d)", i, s.Start(), s.Stop())
		}
		last = s.Stop()
	}
	return nil
}

func createBlueprintsFromManifest(manifest v1.Manifest) ([]*fileBlueprint, error) {
	fileBlueprintsMap := make(map[string]*fileBlueprint)
	for _, l := range manifest.Layers {
		segmentDescriptor, err := filesegment.ParseDescriptor(l)
		if err != nil {
			return nil, fmt.Errorf("unable to parse descriptor: %w", err)
		}
		fr, present := fileBlueprintsMap[segmentDescriptor.Filename()]
		if !present {
			fr = &fileBlueprint{
				Filename: segmentDescriptor.Filename(),
				Segments: make([]*filesegment.Descriptor, 0),
			}
		}
		fr.Segments = append(fr.Segments, segmentDescriptor)
		fileBlueprintsMap[segmentDescriptor.Filename()] = fr
	}
	res := make([]*fileBlueprint, 0)
	for _, v := range fileBlueprintsMap {
		err := v.Validate()
		if err != nil {
			return nil, fmt.Errorf("file blueprint for '%s' failed with: %w", v.Filename, err)
		}
		res = append(res, v)
	}
	return res, nil
}
