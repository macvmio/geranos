package duplicator

import (
	"errors"
	"os/exec"
)

// CloneFile attempts to clone a file using cp with --reflink=auto.
func CloneFile(srcFile, dstFile string) error {
	// Ensure the command is supported on the platform (Linux-specific)
	if _, err := exec.LookPath("cp"); err != nil {
		return errors.New("cp command not found, ensure it's installed and available on PATH")
	}

	// Execute cp with --reflink=auto to attempt efficient cloning
	cmd := exec.Command("cp", "--reflink=auto", srcFile, dstFile)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
