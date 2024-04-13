package dirimage

/*
func TestWrite_ContextCancelledDuringWork(t *testing.T) {

	tempDir, err := os.MkdirTemp("", "write-test-*")
	require.NoErrorf(t, err, "unable to create temp dir: %v", err)
	defer os.RemoveAll(tempDir)

	err = generateRandomFile(filepath.Join(tempDir, "file1.img"), 100)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(tempDir, "testdir1"), 0o777)
	require.NoError(t, err)

	img := empty.Image
	for i := 0; i < 10; i++ {
		layer, err := filesegment.NewLayer(filepath.Join(tempDir, "file1.img"), filesegment.WithRange(int64(i*10), int64(i*10+9)))
		require.NoError(t, err)

		img, err = mutate.Append(img, mutate.Addendum{
			Layer:       layer,
			Annotations: layer.Annotations(),
			MediaType:   layer.GetMediaType(),
		})
		require.NoError(t, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	di := DirImage{
		Image: img,
	}
	err = di.Write(ctx, filepath.Join(tempDir, "testdir1"), WithWorkersCount(2))

	if err == nil || !errors.Is(err, context.Canceled) {
		t.Errorf("Write did not return expected context.Canceled error during work, got: %v", err)
	}
}
*/
