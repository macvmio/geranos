package layout

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/validate"
	"github.com/macvmio/geranos/pkg/dirimage"
	"github.com/macvmio/geranos/pkg/duplicator"
	"github.com/macvmio/geranos/pkg/filesegment"
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
	fmt.Printf("%+v\n", st)
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

	if lm.stats.BytesReadCount.Load() != 2000 { // we read each byte twice to calculate diffID and digest
		t.Fatalf("unexpected number of bytes read: expected %v, got %v", 2000, lm.stats.BytesReadCount.Load())
	}

	destRef, err := name.ParseReference("oci.jarosik.online/testrepo/a:v2")
	require.NoErrorf(t, err, "unable to parse reference %v: %v", destRef, err)

	err = lm.Write(ctx, img1, destRef)
	require.NoErrorf(t, err, "unable to write image %v: %v", destRef, err)

	assert.Equal(t, int64(1000), lm.stats.BytesWrittenCount.Load())
	lm.stats.Clear()

	destRef3, err := name.ParseReference("oci.jarosik.online/testrepo/a:v3")
	require.NoErrorf(t, err, "unable to parse reference %v: %v", destRef3, err)

	err = lm.Write(ctx, img1, destRef3)
	require.NoErrorf(t, err, "unable to write image %v: %v", destRef, err)
	assert.Equal(t, int64(0), lm.stats.BytesWrittenCount.Load())
	assert.Equal(t, int64(1000), lm.stats.BytesReadCount.Load())
	assert.Equal(t, int64(1000), lm.stats.BytesClonedCount.Load())
	assert.Equal(t, int64(100), lm.stats.MatchedSegmentsCount.Load())

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

	if lm.stats.BytesReadCount.Load() != 2000 { // we read each byte twice to calculate diffID and digest
		t.Fatalf("unexpected number of bytes read: expected %v, got %v", 2000, lm.stats.BytesReadCount.Load())
	}

	destRef, err := name.ParseReference("oci.jarosik.online/testrepo/a:v2")
	require.NoErrorf(t, err, "unable to parse reference %v: %v", destRef, err)

	err = lm.Write(ctx, img1, destRef)
	require.NoErrorf(t, err, "unable to write image %v: %v", destRef, err)

	assert.Equal(t, int64(1000), lm.stats.BytesWrittenCount.Load())
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
	assert.Equal(t, int64(20), lm.stats.BytesWrittenCount.Load())
	assert.Equal(t, int64(1020), lm.stats.BytesReadCount.Load())
	assert.Equal(t, int64(1020), lm.stats.BytesClonedCount.Load())
	assert.Equal(t, int64(100), lm.stats.MatchedSegmentsCount.Load())
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

func TestLayoutMapper_ContainsAny(t *testing.T) {
	ref, err := name.ParseReference("example.com/repo/image:tag")
	assert.NoError(t, err)

	// Test case 1: Directory exists
	t.Run("Directory exists", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-directory-*")
		assert.NoError(t, err)
		lm := &Mapper{rootDir: tmpDir}
		defer os.RemoveAll(tmpDir) // Clean up after the test

		dir := filepath.Join(tmpDir, ref.String())
		err = os.MkdirAll(dir, 0755)
		assert.NoError(t, err)

		exists, err := lm.ContainsAny(ref)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	// Test case 2: Directory does not exist
	t.Run("Directory does not exist", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "test-directory-*")
		assert.NoError(t, err)
		lm := &Mapper{rootDir: tmpDir}
		defer os.RemoveAll(tmpDir) // Clean up after the test

		exists, err := lm.ContainsAny(ref)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestLayoutMapper_WriteIfNotPresent(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "layout-mapper-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testRepoDir := path.Join(tempDir, "oci.jarosik.online/testrepo")
	err = os.MkdirAll(path.Join(testRepoDir, "a:v1-origin"), os.ModePerm)
	require.NoErrorf(t, err, "unable to create directory: %v", err)

	randomFileName := path.Join(testRepoDir, "a:v1-origin/disk.img")
	err = generateRandomFile(randomFileName, 123)
	require.NoErrorf(t, err, "unable to generate file: %v", err)
	//hashBefore := hashFromFile(t, randomFileName)
	srcRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v1-origin"))
	require.NoErrorf(t, err, "unable to parse reference: %v", err)
	originImg, err := NewMapper(tempDir).Read(ctx, srcRef)

	// Case 1: Manifests are the same (should not trigger write)
	t.Run("Manifests are the same", func(t *testing.T) {
		// Write the image initially to ensure a local manifest exists
		dstRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v2"))
		require.NoError(t, err)
		lm := NewMapper(tempDir)
		err = lm.Write(ctx, originImg, dstRef)
		require.NoError(t, err)

		// Call WriteIfNotPresent, should skip writing
		lm2 := NewMapper(tempDir)
		err = lm2.WriteIfNotPresent(ctx, originImg, dstRef)
		require.NoError(t, err)

		// Assert that no additional writes occurred (you can check stats or logs)
		assert.Equal(t, int64(123), lm.stats.BytesWrittenCount.Load(), "Expected writes with Write method")
		assert.Equal(t, int64(0), lm2.stats.BytesWrittenCount.Load(), "Expected no additional writes")
	})

	// Case 2: Manifests are different (should trigger write)
	t.Run("Manifests are different", func(t *testing.T) {
		dstRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v3"))
		require.NoError(t, err)
		lm := NewMapper(tempDir)
		err = lm.Write(ctx, originImg, dstRef)
		require.NoError(t, err)

		// Modify the image to create a new manifest
		err = generateRandomFile(randomFileName, 123)
		require.NoErrorf(t, err, "unable to generate file: %v", err)
		updateImage, err := lm.Read(ctx, srcRef)

		// Call WriteIfNotPresent, should perform the write since manifests are different
		lm2 := NewMapper(tempDir)
		err = lm2.WriteIfNotPresent(ctx, updateImage, dstRef)
		require.NoError(t, err)

		// Assert that the image was written to disk
		assert.Equal(t, int64(123), lm2.stats.BytesWrittenCount.Load(), "Expected image write to occur")
	})

	// Case 3: Local manifest does not exist (should trigger write)
	t.Run("Local manifest is missing", func(t *testing.T) {
		dstRef, err := name.ParseReference(fmt.Sprintf("oci.jarosik.online/testrepo/a:v5"))

		// Call WriteIfNotPresent, should perform the write since manifests are different
		lm2 := NewMapper(tempDir)
		err = lm2.WriteIfNotPresent(ctx, originImg, dstRef)
		require.NoError(t, err)

		// Assert that the image was written to disk
		assert.Equal(t, int64(0), lm2.stats.BytesWrittenCount.Load())
		assert.Equal(t, int64(123), lm2.stats.BytesClonedCount.Load())
	})
}

func TestLayoutMapper_Clone_SameFileDifferentNames(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "clone-same-file-*")
	require.NoErrorf(t, err, "unable to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)

	// Setup two directories for Image A and Image B
	fileContent := []byte("same file content")
	images := make([]*dirimage.DirImage, 0)
	for _, x := range []string{"A", "B"} {
		imgDir := path.Join(tempDir, "tmp"+x)
		err = os.MkdirAll(imgDir, os.ModePerm)
		require.NoErrorf(t, err, "unable to create directory for Image A: %v", err)
		err = os.WriteFile(filepath.Join(imgDir, fmt.Sprintf("disk%s.img", x)), fileContent, 0644)
		require.NoErrorf(t, err, "unable to write file for image: %v", err)
		img, err := dirimage.Read(ctx, imgDir, dirimage.WithChunkSize(1))
		require.NoErrorf(t, err, "unable to read image: %v", err)
		images = append(images, img)
	}

	// Write Image A and Image B to the same repo (simulating clone)
	repoDir := path.Join(tempDir, "oci.jarosik.online/testrepo")
	lm := NewMapper(tempDir)

	// Write Image A
	srcRefA, err := name.ParseReference("oci.jarosik.online/testrepo/a:v1")
	require.NoErrorf(t, err, "unable to parse reference for Image A: %v", err)
	err = lm.Write(ctx, images[0], srcRefA)
	require.NoErrorf(t, err, "unable to write Image A: %v", err)

	stats1 := lm.Stats()
	assert.Equal(t, 0, int(stats1.BytesClonedCount))
	assert.Equal(t, len(fileContent), int(stats1.BytesWrittenCount))
	// Write Image B
	lm.stats.Clear()
	srcRefB, err := name.ParseReference("oci.jarosik.online/testrepo/b:v1")
	require.NoErrorf(t, err, "unable to parse reference for Image B: %v", err)
	err = lm.Write(ctx, images[1], srcRefB)
	require.NoErrorf(t, err, "unable to write Image B: %v", err)
	stats2 := lm.Stats()
	assert.Equal(t, len(fileContent), int(stats2.BytesClonedCount))
	assert.Equal(t, 0, int(stats2.BytesWrittenCount))

	// Validate that both images are present with the correct file names
	assert.FileExists(t, filepath.Join(repoDir, "a:v1/diskA.img"))
	assert.FileExists(t, filepath.Join(repoDir, "b:v1/diskB.img"))

	// Validate that the contents of both files are still the same
	hashAfterA := hashFromFile(t, filepath.Join(repoDir, "a:v1/diskA.img"))
	hashAfterB := hashFromFile(t, filepath.Join(repoDir, "b:v1/diskB.img"))
	assert.Equal(t, hashAfterA, hashAfterB, "Both files should still have the same hash after cloning")
}
