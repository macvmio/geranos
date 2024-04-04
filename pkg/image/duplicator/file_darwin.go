package duplicator

import (
	"os/exec"
)

// CloneFile attempts to clone a file using 'cp -c'
func CloneFile(srcFile, dstFile string) error {
	// Execute 'cp -c' to attempt efficient cloning
	cmd := exec.Command("/bin/cp", "-c", srcFile, dstFile)
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
