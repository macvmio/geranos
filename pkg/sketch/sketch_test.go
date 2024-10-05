package sketch

import (
	"context"
	"fmt"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/macvmio/geranos/pkg/dirimage"
	"github.com/macvmio/geranos/pkg/filesegment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultSketchConstructor_ConstructConstruct(t *testing.T) {
	const layersCount = 10
	const testManifestName = ".oci.test.json"
	prepare5CloneCandidatesWith10Layers := func(rootDir string) []*filesegment.Descriptor {
		descriptors := make([]*filesegment.Descriptor, 0)
		for i := 0; i < 5; i++ {
			localDir := filepath.Join(rootDir, fmt.Sprintf("v%d", i))
			err := os.MkdirAll(localDir, os.ModePerm)
			require.NoError(t, err)

			// Create fake files
			err = os.WriteFile(filepath.Join(localDir, "disk.img"), []byte("0123456789"), 0o755)
			require.NoError(t, err)
			err = os.WriteFile(filepath.Join(localDir, "disk2.img"), []byte("0123456789"), 0o755)
			require.NoError(t, err)

			img, err := dirimage.Read(context.Background(), localDir, dirimage.WithChunkSize(1))
			require.NoError(t, err)
			// Convert manifest layers to filesegment.Descriptor
			manifest, err := img.Manifest()
			require.NoError(t, err)
			for _, d := range manifest.Layers {
				fDescriptor, err := filesegment.ParseDescriptor(d)
				require.NoError(t, err)
				descriptors = append(descriptors, fDescriptor)
			}
			manifestBytes, err := img.RawManifest()
			require.NoError(t, err)
			err = os.WriteFile(filepath.Join(localDir, testManifestName), manifestBytes, 0o777)
			require.NoError(t, err)
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
		prepareManifest        func(ds []*filesegment.Descriptor) v1.Manifest
		bytesClonedCount       int64
		matchedSegmentsCount   int64
		prepareCloneCandidates func(rootDir string) []*filesegment.Descriptor
		expectedErr            error
	}{
		{
			name: "successful construct of single recipe",
			prepareManifest: func(ds []*filesegment.Descriptor) v1.Manifest {
				return makeManifest(filesegment.NewDescriptor("disk.img", 0, 1, ds[0].Digest()))
			},
			bytesClonedCount:       2,
			matchedSegmentsCount:   1,
			prepareCloneCandidates: prepare5CloneCandidatesWith10Layers,
			expectedErr:            nil,
		},
		{
			name: "successful construct of all file recipes",
			prepareManifest: func(ds []*filesegment.Descriptor) v1.Manifest {
				return makeManifest(
					filesegment.NewDescriptor("disk.img", 0, 2, ds[1].Digest()),
					filesegment.NewDescriptor("disk.img", 3, 4, ds[2].Digest()),
					filesegment.NewDescriptor("disk2.img", 0, 10, ds[0].Digest()),
				)
			},

			bytesClonedCount:       16,
			matchedSegmentsCount:   3,
			prepareCloneCandidates: prepare5CloneCandidatesWith10Layers,
			expectedErr:            nil,
		},
		{
			name: "successful construct from best clone",
			prepareManifest: func(ds []*filesegment.Descriptor) v1.Manifest {
				return makeManifest(
					filesegment.NewDescriptor("disk.img", 0, 1, ds[0].Digest()),
					filesegment.NewDescriptor("disk.img", 2, 3, ds[1].Digest()),
					filesegment.NewDescriptor("disk.img", 4, 5, ds[2].Digest()),
					filesegment.NewDescriptor("disk.img", 6, 7, ds[3].Digest()),
					filesegment.NewDescriptor("disk.img", 8, 9, ds[4].Digest()),
					filesegment.NewDescriptor("disk.img", 10, 11, ds[5].Digest()),
					filesegment.NewDescriptor("disk.img", 12, 13, ds[6].Digest()),
				)
			},

			bytesClonedCount:       14,
			matchedSegmentsCount:   7,
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
	const testManifestName = ".oci.test.json"
	tests := []struct {
		name           string
		setup          func(rootDir string) // Function to set up the test's file system state
		expectedLength int
		expectedErr    bool // Whether an error is expected
	}{
		{
			name: "single manifest file",
			setup: func(rootDir string) {
				localDir := filepath.Join(rootDir, "directory1")
				err := os.MkdirAll(localDir, os.ModePerm)
				require.NoError(t, err)

				// Create fake files
				err = os.WriteFile(filepath.Join(localDir, "disk.img"), []byte("0123456789"), 0o755)
				require.NoError(t, err)
				err = os.WriteFile(filepath.Join(localDir, "disk2.img"), []byte("0123456789"), 0o755)
				require.NoError(t, err)

				// Use dirimage to read and generate manifest
				img, err := dirimage.Read(context.Background(), localDir, dirimage.WithChunkSize(1))
				require.NoError(t, err)

				manifestBytes, err := img.RawManifest()
				require.NoError(t, err)

				err = os.WriteFile(filepath.Join(localDir, testManifestName), manifestBytes, 0o777)
				require.NoError(t, err)
			},
			expectedLength: 2,
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
				for i := 0; i < 5; i++ {
					localDir := filepath.Join(rootDir, fmt.Sprintf("directory%d", i))
					err := os.MkdirAll(localDir, os.ModePerm)
					require.NoError(t, err)

					// Create fake files
					err = os.WriteFile(filepath.Join(localDir, "disk.img"), []byte("0123456789"), 0o755)
					require.NoError(t, err)
					err = os.WriteFile(filepath.Join(localDir, "disk2.img"), []byte("01234567"), 0o755)
					require.NoError(t, err)
					err = os.WriteFile(filepath.Join(localDir, "disk3.img"), []byte("66666"), 0o755)
					require.NoError(t, err)

					// Use dirimage to read and generate manifest
					img, err := dirimage.Read(context.Background(), localDir, dirimage.WithChunkSize(1))
					require.NoError(t, err)

					manifestBytes, err := img.RawManifest()
					require.NoError(t, err)

					err = os.WriteFile(filepath.Join(localDir, testManifestName), manifestBytes, 0o777)
					require.NoError(t, err)
				}
			},
			expectedLength: 15,
			expectedErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up temporary directory for the test
			rootDir, err := os.MkdirTemp("", "test_find_clone_candidates")
			assert.NoError(t, err)
			defer os.RemoveAll(rootDir) // Cleanup after the test

			// Set up the specific file system state for this test
			tt.setup(rootDir)

			// Call the method under test
			sc := NewSketcher(rootDir, testManifestName)
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

func TestSketchConstructor_ComputeScore(t *testing.T) {
	// Define test cases
	// Define test cases
	newDescriptor := func(hash string) filesegment.Descriptor {
		return *filesegment.NewDescriptor("test", 0, 0, v1.Hash{Algorithm: "sha256", Hex: hash})
	}

	tests := []struct {
		name               string
		segmentDescriptors map[string]filesegment.Descriptor
		descriptors        []filesegment.Descriptor
		expectedScore      int
	}{
		{
			name: "No match - different hash",
			segmentDescriptors: map[string]filesegment.Descriptor{
				"sha256:hash1": newDescriptor("hash1"),
			},
			descriptors: []filesegment.Descriptor{
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash2"}),
			},
			expectedScore: 0,
		},
		{
			name: "Single match",
			segmentDescriptors: map[string]filesegment.Descriptor{
				"sha256:hash1": newDescriptor("hash1"),
			},
			descriptors: []filesegment.Descriptor{
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash1"}),
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
			descriptors: []filesegment.Descriptor{
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash2"}),
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash1"}),
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash3"}),
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
			descriptors: []filesegment.Descriptor{
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash5"}),
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash3"}),
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash1"}),
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash11"}),
			},
			expectedScore: 3,
		},
		{
			name: "Mismatch and match",
			segmentDescriptors: map[string]filesegment.Descriptor{
				"sha256:hash1": newDescriptor("hash1"),
				"sha256:hash4": newDescriptor("hash4"),
			},
			descriptors: []filesegment.Descriptor{
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash1"}),
				*filesegment.NewDescriptor("testfile", 0, 0, v1.Hash{Algorithm: "sha256", Hex: "hash2"}),
			},
			expectedScore: 1,
		},
		{
			name: "Duplicate Descriptors",
			segmentDescriptors: map[string]filesegment.Descriptor{
				"sha256:hash1": newDescriptor("hash1"),
				"sha256:hash2": newDescriptor("hash2"),
				"sha256:hash3": newDescriptor("hash3"),
			},
			descriptors: []filesegment.Descriptor{
				newDescriptor("hash1"),
				newDescriptor("hash1"), // Duplicate
				newDescriptor("hash2"),
				newDescriptor("hash2"), // Duplicate
				newDescriptor("hash3"),
			},
			expectedScore: 3, // Should count each unique digest only once
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
