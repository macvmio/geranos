package image

import (
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSketchConstructor_ConstructConstruct(t *testing.T) {
	const layersCount = 10
	prepare5CloneCandidatesWith10Layers := func(rootDir string) []*v1.Descriptor {
		descriptors := make([]*v1.Descriptor, 0)
		for i := 0; i < 5; i++ {
			img, err := random.Image(1024, layersCount)
			require.NoError(t, err)
			manifestBytes, err := img.RawManifest()
			require.NoError(t, err)
			localDir := filepath.Join(rootDir, fmt.Sprintf("v%d", i))
			err = os.MkdirAll(localDir, 0o777)
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
			err = os.WriteFile(filepath.Join(localDir, LocalManifestFilename), manifestBytes, 0o777)
		}
		return descriptors
	}
	type Seg struct {
		Start  int64
		Stop   int64
		Digest v1.Hash
	}
	makeFR := func(filename string, seg ...Seg) fileRecipe {
		segs := make([]fileSegmentRecipe, 0)
		for _, s := range seg {
			segs = append(segs, fileSegmentRecipe{
				Filename: filename,
				Start:    s.Start,
				Stop:     s.Stop,
				Digest:   s.Digest,
			})
		}
		return fileRecipe{
			Filename: filename,
			Segments: segs,
		}
	}

	tests := []struct {
		name                   string
		prepareFileRecipes     func(ds []*v1.Descriptor) []*fileRecipe
		expectedStats          Statistics
		prepareCloneCandidates func(rootDir string) []*v1.Descriptor
		expectedErr            error
	}{
		{
			name: "successful construct of single recipe",
			prepareFileRecipes: func(ds []*v1.Descriptor) []*fileRecipe {
				fr := makeFR("disk.img", Seg{0, 1, ds[0].Digest})
				return []*fileRecipe{&fr}
			},
			expectedStats: Statistics{
				BytesClonedCount:      2,
				MatchingSegmentsCount: 1,
			},
			prepareCloneCandidates: prepare5CloneCandidatesWith10Layers,
			expectedErr:            nil,
		},
		{
			name: "successful construct of all file recipes",
			prepareFileRecipes: func(ds []*v1.Descriptor) []*fileRecipe {
				fr1 := makeFR("disk.img",
					Seg{0, 2, ds[1].Digest},
					Seg{3, 4, ds[2].Digest})
				fr2 := makeFR("disk2.img", Seg{0, 10, ds[0].Digest})
				return []*fileRecipe{&fr1, &fr2}
			},
			expectedStats: Statistics{
				BytesClonedCount:      16,
				MatchingSegmentsCount: 3,
			},
			prepareCloneCandidates: prepare5CloneCandidatesWith10Layers,
			expectedErr:            nil,
		},
		{
			name: "successful construct from best clone",
			prepareFileRecipes: func(ds []*v1.Descriptor) []*fileRecipe {
				fr1 := makeFR("disk.img",
					Seg{0, 1, ds[0].Digest},
					Seg{2, 3, ds[1*layersCount+0].Digest},
					Seg{4, 5, ds[2*layersCount+0].Digest},
					Seg{6, 7, ds[2*layersCount+1].Digest},
					Seg{8, 9, ds[2*layersCount+2].Digest},
					Seg{10, 11, ds[3*layersCount+0].Digest},
					Seg{12, 13, ds[3*layersCount+1].Digest},
				)
				return []*fileRecipe{&fr1}
			},
			expectedStats: Statistics{
				BytesClonedCount:      14,
				MatchingSegmentsCount: 3,
			},
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

			sc := defaultSketchConstructor{
				rootDirectory: rootDir,
				stats:         Statistics{},
			}

			// Call the prepare function to set up clone candidates
			descriptors := tt.prepareCloneCandidates(sc.rootDirectory)

			stats, err := sc.Construct(filepath.Join(rootDir, "some/dir"), tt.prepareFileRecipes(descriptors))

			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStats, stats)
			}
		})
	}
}

func TestSketchConstructor_FindCloneCandidates(t *testing.T) {
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
				err := os.MkdirAll(dirPath, 0755)
				require.NoError(t, err)
				manifestPath := filepath.Join(dirPath, LocalManifestFilename)
				err = os.WriteFile(manifestPath, []byte("{}"), 0644) // Write an empty JSON object as a placeholder
				require.NoError(t, err)
			},
			expectedLength: 1,
			expectedErr:    false,
		},
		{
			name: "no manifest files",
			setup: func(rootDir string) {
				err := os.MkdirAll(filepath.Join(rootDir, "directory1"), 0755)
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
					err := os.MkdirAll(dirPath, 0755)
					require.NoError(t, err)
					err = os.WriteFile(filepath.Join(dirPath, LocalManifestFilename), []byte("{}"), 0644)
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
			sc := defaultSketchConstructor{rootDirectory: rootDir}
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
	tests := []struct {
		name           string
		segmentRecipes map[string]*fileSegmentRecipe
		descriptors    []*v1.Descriptor
		expectedScore  int
	}{
		{
			name: "No match - different hash",
			segmentRecipes: map[string]*fileSegmentRecipe{
				"hash1": {Digest: v1.Hash{Hex: "hash1"}},
			},
			descriptors: []*v1.Descriptor{
				{Digest: v1.Hash{Hex: "hash2"}},
			},
			expectedScore: 0,
		},
		{
			name: "Single match",
			segmentRecipes: map[string]*fileSegmentRecipe{
				"sha256:hash1": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
			},
			descriptors: []*v1.Descriptor{
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
			},
			expectedScore: 1,
		},
		{
			name: "Multiple matches",
			segmentRecipes: map[string]*fileSegmentRecipe{
				"sha256:hash1": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
				"sha256:hash2": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash2"}},
				"sha256:hash3": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash3"}},
			},
			descriptors: []*v1.Descriptor{
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash2"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash3"}},
			},
			expectedScore: 3,
		},
		{
			name: "Multiple matches - interleaving",
			segmentRecipes: map[string]*fileSegmentRecipe{
				"sha256:hash0": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash0"}},
				"sha256:hash1": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
				"sha256:hash2": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash2"}},
				"sha256:hash3": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash3"}},
				"sha256:hash4": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash4"}},
				"sha256:hash5": {Digest: v1.Hash{Algorithm: "sha256", Hex: "hash5"}},
			},
			descriptors: []*v1.Descriptor{
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash5"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash3"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash1"}},
				{Digest: v1.Hash{Algorithm: "sha256", Hex: "hash11"}},
			},
			expectedScore: 3,
		},
		{
			name: "Mismatch and match",
			segmentRecipes: map[string]*fileSegmentRecipe{
				"sha256:hash1": {Digest: v1.Hash{Hex: "hash1"}},
				"sha256:hash4": {Digest: v1.Hash{Hex: "hash4"}},
			},
			descriptors: []*v1.Descriptor{
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
			sc := defaultSketchConstructor{}
			cc := cloneCandidate{
				descriptors: tt.descriptors,
			}
			// Pass segmentRecipes map instead of sortedSegments slice
			score := sc.computeScore(tt.segmentRecipes, &cc)
			assert.Equal(t, tt.expectedScore, score)
		})
	}
}
