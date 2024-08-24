package layout

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/mobileinf/geranos/pkg/dirimage"
	"github.com/mobileinf/geranos/pkg/duplicator"
	"github.com/mobileinf/geranos/pkg/filesegment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path"
	"path/filepath"
	"testing"
)

func hashFromFile(t *testing.T, filename string) string {
	f, err := os.Open(filename)
	require.NoErrorf(t, err, "unexpected error while opening a file, got %v", err)
	defer f.Close()
	h, _, err := v1.SHA256(f)
	require.NoErrorf(t, err, "unable to calculate SHA256, got %v", err)
	return h.Hex
}

func TestLayoutMapper_Read(t *testing.T) {
	lm := NewMapper("testdata")
	ref, err := name.ParseReference("vm1")
	require.NoErrorf(t, err, "unable to parse reference: %v", err)
	img, err := lm.Read(context.Background(), ref)
	require.NoErrorf(t, err, "unable to read layout from disk: %v", err)
	err = validate.Image(img, validate.Fast)
	require.NoErrorf(t, err, "image validation error: %v", err)
	st := lm.Stats()
	fmt.Printf("%#v\n", st)
}

func TestLayoutMapper_Read_VariousChunkSizes(t *testing.T) {
	hashBefore := hashFromFile(t, "testdata/vm1/disk.blob")
	tempDir, err := os.MkdirTemp("", "oci-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	lmDst := NewMapper(tempDir)
	srcRef, err := name.ParseReference("vm1")
	require.NoErrorf(t, err, "unable to parse source reference: %v", err)
	dstRef, err := name.ParseReference("vmdst")
	require.NoErrorf(t, err, "unable to parse destination reference: %v", err)

	for chunkSize := int64(1); chunkSize < 10; chunkSize++ {
		lmSrc := NewMapper("testdata", dirimage.WithChunkSize(chunkSize))
		img, err := lmSrc.Read(context.Background(), srcRef)
		require.NoErrorf(t, err, "unable to read image: %v", err)
		err = validate.Image(img, validate.Fast)
		if err != nil || img == nil {
			t.Fatalf("img is not correct: %v", err)
		}
		err = lmDst.Write(context.Background(), img, dstRef)
		require.NoErrorf(t, err, "unable to write image to destination: %v", err)
		hashAfter := hashFromFile(t, filepath.Join(tempDir, dstRef.String(), "disk.blob"))
		if hashBefore != hashAfter {
			t.Fatalf("hashes differ: expected: %v, got: %v", hashBefore, hashAfter)
		}
	}
	st := lmDst.Stats()
	fmt.Printf("%#v\n", st)
	if st.BytesWrittenCount != 102 {
		t.Fatalf("unexpected number of bytes written: expected %d got %d", 918, st.BytesWrittenCount)
	}
}

// GenerateRandomFile creates a file of the specified size filled with random bytes
// using io.CopyN for efficient copying.
func generateRandomFile(filename string, size int64) error {
	// Open a file for writing, creating it with 0666 permissions if it does not exist
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer file.Close()

	// Copy the specified amount of random data to the file
	// rand.Reader is a global, shared instance of a cryptographically secure random number generator
	if _, err := io.CopyN(file, rand.Reader, size); err != nil {
		return fmt.Errorf("error copying random data to file: %w", err)
	}

	return nil
}

func appendRandomBytesToFile(filename string, numBytes int64) error {
	// Open file in append mode. If file doesn't exist, create it with permissions 0644
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Use crypto/rand.Reader as the source of random bytes
	// and copy the specified number of bytes to the file
	_, err = io.CopyN(file, rand.Reader, numBytes)
	if err != nil {
		return err
	}

	return nil
}

func TestLayoutMapper_Write_MustOptimizeDiskSpace(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "optimized-disk-*")
	if err != nil {
		t.Fatalf("unable to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	testRepoDir := path.Join(tempDir, "oci.jarosik.online/testrepo")
	err = os.MkdirAll(path.Join(testRepoDir, "a:v1"), os.ModePerm)
	require.NoErrorf(t, err, "unable to create directory: %v", err)
	optimalRepoDir := path.Join(tempDir, "oci.jarosik.online/optimalrepo")
	err = os.MkdirAll(path.Join(optimalRepoDir, "a:v1"), os.ModePerm)
	require.NoErrorf(t, err, "unable to create directory: %v", err)

	MB := int64(1024 * 1024)
	chunkSize := MB
	randomFileName := path.Join(testRepoDir, "a:v1/disk.img")
	err = generateRandomFile(randomFileName, 32*MB)
	require.NoErrorf(t, err, "unable to generate file: %v", err)
	hashBefore := hashFromFile(t, randomFileName)
	srcRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v1"))
	require.NoErrorf(t, err, "unable to parse reference: %v", err)
	lm := NewMapper(tempDir, dirimage.WithChunkSize(chunkSize))
	img1, err := lm.Read(ctx, srcRef)
	require.NoErrorf(t, err, "unable to read disk image: %v", err)
	for i := 2; i < 12; i++ {
		dir := fmt.Sprintf("oci.jarosik.online/testrepo/a:v%d", i)
		r, err := name.ParseReference(dir)
		require.NoErrorf(t, err, "unable to parse reference %d: %v", i, err)
		err = lm.Write(ctx, img1, r)
		require.NoErrorf(t, err, "unable to write image %d: %v", i, err)
		err = duplicator.CloneDirectory(path.Join(testRepoDir, "a:v1"), path.Join(optimalRepoDir, fmt.Sprintf("a:v%d", i)), false)
		require.NoErrorf(t, err, "unable to clone directory: %v", err)
		assert.Equal(t, hashBefore, hashFromFile(t, filepath.Join(testRepoDir, fmt.Sprintf("a:v%d", i), "disk.img")))
	}
	for _, repo := range []string{testRepoDir, optimalRepoDir} {
		diskUsage, err := DirectoryDiskUsage(testRepoDir)
		require.NoErrorf(t, err, "unable to calculate disk usage: %v", err)
		fmt.Printf("[%v] total disk used: %v\n", repo, diskUsage)
	}
	st := lm.Stats()
	fmt.Printf("stats: %#v\n", st)
	if 32*MB != st.BytesWrittenCount {
		t.Fatalf("unexpected number of bytes written")
	}
}

func TestLayoutMapper_Write_MustAvoidWritingSameContent(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "content-matches-*")
	require.NoErrorf(t, err, "unable to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)
	testRepoDir := path.Join(tempDir, "oci.jarosik.online/testrepo")
	err = os.MkdirAll(path.Join(testRepoDir, "a:v1"), os.ModePerm)
	require.NoErrorf(t, err, "unable to create directory: %v", err)
	const chunkSize = 10
	lm := NewMapper(tempDir, dirimage.WithChunkSize(chunkSize))
	err = generateRandomFile(path.Join(testRepoDir, "a:v1/disk.img"), 100*chunkSize)
	require.NoErrorf(t, err, "unable to generate file: %v", err)

	srcRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v1"))
	require.NoErrorf(t, err, "unable to parse reference: %v", err)
	beforeHash := hashFromFile(t, path.Join(tempDir, "oci.jarosik.online/testrepo/a:v1/disk.img"))
	img1, err := lm.Read(ctx, srcRef)
	require.NoErrorf(t, err, "unable to read disk image: %v", err)

	if lm.stats.BytesReadCount != 2000 { // we read each byte twice to calculate diffID and digest
		t.Fatalf("unexpected number of bytes read: expected %v, got %v", 2000, lm.stats.BytesReadCount)
	}

	destRef, err := name.ParseReference("oci.jarosik.online/testrepo/a:v2")
	require.NoErrorf(t, err, "unable to parse reference %v: %v", destRef, err)

	err = lm.Write(ctx, img1, destRef)
	require.NoErrorf(t, err, "unable to write image %v: %v", destRef, err)

	assert.Equal(t, int64(1000), lm.stats.BytesWrittenCount)
	lm.stats.Clear()

	destRef3, err := name.ParseReference("oci.jarosik.online/testrepo/a:v3")
	require.NoErrorf(t, err, "unable to parse reference %v: %v", destRef3, err)

	err = lm.Write(ctx, img1, destRef3)
	require.NoErrorf(t, err, "unable to write image %v: %v", destRef, err)
	assert.Equal(t, int64(0), lm.stats.BytesWrittenCount)
	assert.Equal(t, int64(1000), lm.stats.BytesReadCount)
	assert.Equal(t, int64(1000), lm.stats.BytesClonedCount)
	assert.Equal(t, int64(100), lm.stats.MatchedSegmentsCount)

	afterHash := hashFromFile(t, path.Join(tempDir, "oci.jarosik.online/testrepo/a:v3/disk.img"))
	assert.Equal(t, beforeHash, afterHash)
}

func TestLayoutMapper_Write_MustOnlyWriteContentThatDiffersFromAlreadyWritten(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "content-matches-*")
	require.NoErrorf(t, err, "unable to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)
	testRepoDir := path.Join(tempDir, "oci.jarosik.online/testrepo")
	err = os.MkdirAll(path.Join(testRepoDir, "a:v1"), os.ModePerm)
	require.NoErrorf(t, err, "unable to create directory: %v", err)
	const chunkSize = 10
	lm := NewMapper(tempDir, dirimage.WithChunkSize(chunkSize))
	randomFilename := path.Join(testRepoDir, "a:v1/disk.img")
	err = generateRandomFile(randomFilename, 100*chunkSize)
	require.NoErrorf(t, err, "unable to generate file: %v", err)

	srcRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v1"))
	require.NoErrorf(t, err, "unable to parse reference: %v", err)
	img1, err := lm.Read(ctx, srcRef)
	require.NoErrorf(t, err, "unable to read disk image: %v", err)

	if lm.stats.BytesReadCount != 2000 { // we read each byte twice to calculate diffID and digest
		t.Fatalf("unexpected number of bytes read: expected %v, got %v", 2000, lm.stats.BytesReadCount)
	}

	destRef, err := name.ParseReference("oci.jarosik.online/testrepo/a:v2")
	require.NoErrorf(t, err, "unable to parse reference %v: %v", destRef, err)

	err = lm.Write(ctx, img1, destRef)
	require.NoErrorf(t, err, "unable to write image %v: %v", destRef, err)

	assert.Equal(t, int64(1000), lm.stats.BytesWrittenCount)
	lm.stats.Clear()

	// Here "testrepo/a:v2" contains .oci.manifest.json, and is the same as generated file

	err = appendRandomBytesToFile(randomFilename, 21)
	require.NoError(t, err)
	l1, err := filesegment.NewLayer(randomFilename, filesegment.WithRange(1000, 1009))
	require.NoError(t, err)
	l2, err := filesegment.NewLayer(randomFilename, filesegment.WithRange(1010, 1019))
	require.NoError(t, err)
	img3, err := mutate.Append(img1, mutate.Addendum{
		Layer:       l1,
		Annotations: l1.Annotations(),
		MediaType:   filesegment.MediaType,
	}, mutate.Addendum{
		Layer:       l2,
		Annotations: l2.Annotations(),
		MediaType:   filesegment.MediaType,
	})
	require.NoError(t, err)

	destRef, err = name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v3"))
	require.NoErrorf(t, err, "unable to parse reference %v: %v", destRef, err)
	err = lm.Write(ctx, img3, destRef)
	require.NoErrorf(t, err, "unable to write image %v: %v", destRef, err)
	assert.Equal(t, int64(20), lm.stats.BytesWrittenCount)
	assert.Equal(t, int64(1020), lm.stats.BytesReadCount)
	assert.Equal(t, int64(1020), lm.stats.BytesClonedCount)
	assert.Equal(t, int64(100), lm.stats.MatchedSegmentsCount)
}

func TestLayoutMapper_Write_MultipleConcurrentWorkers(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "content-matches-*")
	require.NoErrorf(t, err, "unable to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)
	testRepoDir := path.Join(tempDir, "oci.jarosik.online/testrepo")
	err = os.MkdirAll(path.Join(testRepoDir, "a:v1"), os.ModePerm)
	require.NoErrorf(t, err, "unable to create directory: %v", err)
	logF := func(fmt string, argv ...any) {}
	const chunkSize = 11
	lm := NewMapper(tempDir, dirimage.WithChunkSize(chunkSize), dirimage.WithLogFunction(logF))
	err = generateRandomFile(path.Join(testRepoDir, "a:v1/disk.img"), 200*chunkSize)
	require.NoErrorf(t, err, "unable to generate file: %v", err)

	srcRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v1"))
	require.NoErrorf(t, err, "unable to parse reference: %v", err)
	beforeHash := hashFromFile(t, path.Join(tempDir, "oci.jarosik.online/testrepo/a:v1/disk.img"))
	img1, err := lm.Read(ctx, srcRef)
	require.NoErrorf(t, err, "unable to read disk image: %v", err)

	for workersCount := 1; workersCount < 10; workersCount++ {
		t.Run(fmt.Sprintf("Write-with-%d-workers", workersCount), func(t *testing.T) {

			lm2 := NewMapper(tempDir, dirimage.WithChunkSize(chunkSize), dirimage.WithWorkersCount(workersCount), dirimage.WithLogFunction(logF))
			dstRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v%d", workersCount))
			require.NoError(t, err)
			err = lm2.Write(ctx, img1, dstRef)
			require.NoError(t, err)
			afterHash := hashFromFile(t, path.Join(tempDir, dstRef.String(), "disk.img"))
			assert.Equal(t, beforeHash, afterHash)
		})
	}
}

func TestLayoutMapper_Write_MustOverwriteBiggerFileIfAlreadyExist(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "content-matches-*")
	require.NoErrorf(t, err, "unable to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)
	testRepoDir := path.Join(tempDir, "oci.jarosik.online/testrepo")
	err = os.MkdirAll(path.Join(testRepoDir, "a:v1"), os.ModePerm)
	require.NoErrorf(t, err, "unable to create directory: %v", err)
	logF := func(fmt string, argv ...any) {}
	const chunkSize = 5
	lm := NewMapper(tempDir, dirimage.WithChunkSize(chunkSize), dirimage.WithLogFunction(logF))
	err = generateRandomFile(path.Join(testRepoDir, "a:v1/disk.img"), 10*chunkSize)
	require.NoErrorf(t, err, "unable to generate file: %v", err)

	srcRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v1"))
	require.NoErrorf(t, err, "unable to parse reference: %v", err)
	beforeHash := hashFromFile(t, path.Join(tempDir, "oci.jarosik.online/testrepo/a:v1/disk.img"))
	img1, err := lm.Read(ctx, srcRef)
	require.NoErrorf(t, err, "unable to read disk image: %v", err)

	dstRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v2"))
	err = lm.Write(ctx, img1, dstRef)
	require.NoError(t, err)
	hash2 := hashFromFile(t, path.Join(tempDir, "oci.jarosik.online/testrepo/a:v2/disk.img"))
	assert.Equal(t, beforeHash, hash2)

	err = appendRandomBytesToFile(path.Join(tempDir, "oci.jarosik.online/testrepo/a:v2/disk.img"), 11)
	require.NoError(t, err)
	hash3 := hashFromFile(t, path.Join(tempDir, "oci.jarosik.online/testrepo/a:v2/disk.img"))
	require.NotEqual(t, beforeHash, hash3)

	err = lm.Write(ctx, img1, dstRef)
	require.NoError(t, err)
	hash4 := hashFromFile(t, path.Join(tempDir, "oci.jarosik.online/testrepo/a:v2/disk.img"))
	assert.Equal(t, beforeHash, hash4)
}
