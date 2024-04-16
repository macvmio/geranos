package sketch

import (
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tomekjarosik/geranos/pkg/filesegment"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSketchConstructor_ConstructConstruct(t *testing.T) {
	const layersCount = 10
	const testManifestName = ".oci.test.json"
	prepare5CloneCandidatesWith10Layers := func(rootDir string) []*v1.Descriptor {
		descriptors := make([]*v1.Descriptor, 0)
		for i := 0; i < 5; i++ {
			img, err := random.Image(1024, layersCount)
			require.NoError(t, err)
			manifestBytes, err := img.RawManifest()
			require.NoError(t, err)
			localDir := filepath.Join(rootDir, fmt.Sprintf("v%d", i))
			err = os.MkdirAll(localDir, os.ModePerm)
			require.NoError(t, err)

			require.NoError(t, err)
			manifest, err := img.Manifest()
			require.NoError(t, err)
			// Create fake file
			err = os.WriteFile(filepath.Join(localDir, "disk.img"), []byte("0123456789"), 0o755)
			require.NoError(t, err)
			// Create fake file
			err = os.WriteFile(filepath.Join(localDir, "disk2.img"), []byte("0123456789"), 0o755)
			require.NoError(t, err)
			for _, d := range manifest.Layers {
				descriptors = append(descriptors, &d)
			}
			err = os.WriteFile(filepath.Join(localDir, testManifestName), manifestBytes, 0o777)
		}
		return descriptors
	}

	makeManifest := func(seg ...*filesegment.Descriptor) v1.Manifest {
		m := v1.Manifest{Layers: make([]v1.Descriptor, 0)}
		for _, s := range seg {
			m.Layers = append(m.Layers, v1.Descriptor{
				MediaType:   s.MediaType(),
				Size:        0,
				Digest:      s.Digest(),
				Annotations: s.Annotations(),
			})
		}
		return m
	}

	tests := []struct {
		name                   string
		prepareManifest        func(ds []*v1.Descriptor) v1.Manifest
		bytesClonedCount       int64
		matchedSegmentsCount   int64
		prepareCloneCandidates func(rootDir string) []*v1.Descriptor
		expectedErr            error
	}{
		{
			name: "successful construct of single recipe",
			prepareManifest: func(ds []*v1.Descriptor) v1.Manifest {
				return makeManifest(filesegment.NewDescriptor("disk.img", 0, 1, ds[0].Digest))
			},
			bytesClonedCount:       2,
			matchedSegmentsCount:   1,
			prepareCloneCandidates: prepare5CloneCandidatesWith10Layers,
			expectedErr:            nil,
		},
		{
			name: "successful construct of all file recipes",
			prepareManifest: func(ds []*v1.Descriptor) v1.Manifest {
				return makeManifest(
					filesegment.NewDescriptor("disk.img", 0, 2, ds[1].Digest),
					filesegment.NewDescriptor("disk.img", 3, 4, ds[2].Digest),
					filesegment.NewDescriptor("disk2.img", 0, 10, ds[0].Digest),
				)
			},

			bytesClonedCount:     16,
			matchedSegmentsCount: 3,

			prepareCloneCandidates: prepare5CloneCandidatesWith10Layers,
			expectedErr:            nil,
		},
		{
			name: "successful construct from best clone",
			prepareManifest: func(ds []*v1.Descriptor) v1.Manifest {
				/*fr1 := makeFR("disk.img",
					Seg{0, 1, ds[0].Digest},
					Seg{2, 3, ds[1*layersCount+0].Digest},
					Seg{4, 5, ds[2*layersCount+0].Digest},
					Seg{6, 7, ds[2*layersCount+1].Digest},
					Seg{8, 9, ds[2*layersCount+2].Digest},
					Seg{10, 11, ds[3*layersCount+0].Digest},
					Seg{12, 13, ds[3*layersCount+1].Digest},
				)*/
				return makeManifest(
					filesegment.NewDescriptor("disk.img", 0, 1, ds[0].Digest),
					filesegment.NewDescriptor("disk.img", 2, 3, ds[1*layersCount+0].Digest),
					filesegment.NewDescriptor("disk.img", 4, 5, ds[2*layersCount+0].Digest),
					filesegment.NewDescriptor("disk.img", 6, 7, ds[2*layersCount+1].Digest),
					filesegment.NewDescriptor("disk.img", 8, 9, ds[2*layersCount+2].Digest),
					filesegment.NewDescriptor("disk.img", 10, 11, ds[3*layersCount+0].Digest),
					filesegment.NewDescriptor("disk.img", 12, 13, ds[3*layersCount+1].Digest),
				)
			},

			bytesClonedCount:     14,
			matchedSegmentsCount: 3,

			prepareCloneCandidates: prepare5CloneCandidatesWith10Layers,
			expectedErr:            nil,
		},
		// Add more test cases as needed...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootDir, err := os.MkdirTemp("", "sketch_construct")
			assert.NoError(t, err)
			defer os.RemoveAll(rootDir) // Cleanup after the test

			sc := NewSketcher(rootDir, testManifestName)

			// Call the prepare function to set up clone candidates
			descriptors := tt.prepareCloneCandidates(sc.rootDirectory)
			manifest := tt.prepareManifest(descriptors)

			bytesClonedCount, matchedSegmentsCount, err := sc.Sketch(filepath.Join(rootDir, "some/dir"), manifest)

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.bytesClonedCount, bytesClonedCount)
				assert.Equal(t, tt.matchedSegmentsCount, matchedSegmentsCount)
			}
		})
	}
}

func TestSketchConstructor_FindCloneCandidates(t *testing.T) {
	const localManifestFile = ".oci.test.json"
	tests := []struct {
		name           string
		setup          func(rootDir string) // Function to setup the test's file system state
		expectedLength int
		expectedErr    bool // Whether an error is expected
	}{
		{
			name: "single manifest file",
			setup: func(rootDir string) {
				// Create a directory structure with one manifest
				dirPath := filepath.Join(rootDir, "directory1")
				err := os.MkdirAll(dirPath, os.ModePerm)
				require.NoError(t, err)
				manifestPath := filepath.Join(dirPath, localManifestFile)
				err = os.WriteFile(manifestPath, []byte("{}"), 0644) // Write an empty JSON object as a placeholder
				require.NoError(t, err)
			},
			expectedLength: 1,
			expectedErr:    false,
		},
		{
			name: "no manifest files",
			setup: func(rootDir string) {
				err := os.MkdirAll(filepath.Join(rootDir, "directory1"), os.ModePerm)
				require.NoError(t, err)
			},
			expectedLength: 0,
			expectedErr:    false,
		},
		{
			name: "multiple manifest files",
			setup: func(rootDir string) {
				// Create a directory structure with one manifest
				for i := 0; i < 5; i++ {
					dirPath := filepath.Join(rootDir, fmt.Sprintf("directory%d", i))
					err := os.MkdirAll(dirPath, os.ModePerm)
					require.NoError(t, err)
					err = os.WriteFile(filepath.Join(dirPath, localManifestFile), []byte("{}"), 0644)
					require.NoError(t, err)
					err = os.WriteFile(filepath.Join(dirPath, "disk.img"), []byte(""), 0644)
					require.NoError(t, err)
				}
			},
			expectedLength: 5,
			expectedErr:    false,
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temporary directory for the test
			rootDir, err := os.MkdirTemp("", "test_find_clone_candidates")
			assert.NoError(t, err)
			defer os.RemoveAll(rootDir) // Cleanup after the test

			// Setup the specific file system state for this test
			tt.setup(rootDir)

			// Call the method under test
			sc := NewSketcher(rootDir, localManifestFile)
			candidates, err := sc.findCloneCandidates()

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, candidates, tt.expectedLength)
			}
		})
	}
}

// TestComputeScore tests the computeScore method of defaultSketchConstructor for various scenarios.
func TestSketchConstructor_ComputeScore(t *testing.T) {
	// Define test cases
	newDescriptor := func(hash string) filesegment.Descriptor {
		return *filesegment.NewDescriptor("test", 0, 0, v1.Hash{Algorithm: "sha256", Hex: hash})
	}
	tests := []struct {
		name               string
		segmentDescriptors map[string]filesegment.Descriptor
		descriptors        []v1.Descriptor
		expectedScore      int
	}{
		{
			name: "No match - different hash",
			segmentDescriptors: map[string]filesegment.Descriptor{
				"hash1": newDescriptor("hash1"),
			},
			descriptors: []v1.Descriptor{
				{Digest: v1.Hash{Hex: "hash2"}},
			},
			expectedScore: 0,
		},
		{
			name: "Single match",
			segmentDescriptors: map[string]filesegment.Descriptor{
				"sha256:hash1": newDescriptor("hash1"),
			},
			descriptors: []v1.Descriptor{
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
			},
			expectedScore: 1,
		},
		{
			name: "Multiple matches",
			segmentDescriptors: map[string]filesegment.Descriptor{
				"sha256:hash1": newDescriptor("hash1"),
				"sha256:hash2": newDescriptor("hash2"),
				"sha256:hash3": newDescriptor("hash3"),
			},
			descriptors: []v1.Descriptor{
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash2"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash3"}},
			},
			expectedScore: 3,
		},
		{
			name: "Multiple matches - interleaving",
			segmentDescriptors: map[string]filesegment.Descriptor{
				"sha256:hash0": newDescriptor("hash0"),
				"sha256:hash1": newDescriptor("hash1"),
				"sha256:hash2": newDescriptor("hash2"),
				"sha256:hash3": newDescriptor("hash3"),
				"sha256:hash4": newDescriptor("hash4"),
				"sha256:hash5": newDescriptor("hash5"),
			},
			descriptors: []v1.Descriptor{
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash5"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash3"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash11"}},
			},
			expectedScore: 3,
		},
		{
			name: "Mismatch and match",
			segmentDescriptors: map[string]filesegment.Descriptor{
				"sha256:hash1": newDescriptor("hash1"),
				"sha256:hash4": newDescriptor("hash4"),
			},
			descriptors: []v1.Descriptor{
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash2"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash3"}},
			},
			expectedScore: 1,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := Sketcher{}
			cc := cloneCandidate{
				descriptors: tt.descriptors,
			}
			// Pass segmentDescriptors map instead of sortedSegments slice
			score := sc.computeScore(tt.segmentDescriptors, &cc)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}
