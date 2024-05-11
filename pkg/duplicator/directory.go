package duplicator

import (
	"fmt"
	"os"
	"path/filepath"
)

func CloneDirectory(srcDir, dstDir string, recursive bool) error {
	// Read the contents of the source directory
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("unable to read dir '%v': %w", srcDir, err)
	}

	err = os.MkdirAll(dstDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create dst directory '%v': %w", dstDir, err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			if recursive {
				// If the entry is a directory, recursively clone it
				err = CloneDirectory(srcPath, dstPath, recursive)
				if err != nil {
					return fmt.Errorf("failed to clone src directory '%v' to destination '%v': %w", srcPath, dstPath, err)
				}
			}
		} else {
			// If the entry is a file, clone it
			err = CloneFile(srcPath, dstPath)
			if err != nil {
				return fmt.Errorf("failed to clone src file '%v' to destination '%v': %w", srcPath, dstPath, err)
			}
		}
	}

	return nil
}
