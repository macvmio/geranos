package layout

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// DirectoryDiskUsage returns the disk usage of the specified directory.
func DirectoryDiskUsage(path string) (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// On Windows, we use PowerShell with a custom command to get the folder size
		psCmd := fmt.Sprintf("Get-ChildItem -Recurse '%s' | Measure-Object -Property Length -Sum", path)
		cmd = exec.Command("powershell", "-NoProfile", "-Command", psCmd, "| Select-Object Sum | Format-Table -HideTableHeaders")
	case "darwin", "linux":
		// On macOS and Linux, we can use the 'du' command directly
		cmd = exec.Command("du", "-sh", path)
	default:
		return "", fmt.Errorf("unsupported platform")
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	result := strings.TrimSpace(out.String())

	if runtime.GOOS == "windows" {
		// Parse PowerShell output to get the size in a similar format to du -sh
		lines := strings.Split(result, "\n")
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		sizeInBytes := strings.Fields(lastLine)[0]
		size, err := formatBytesToHumanReadable(sizeInBytes)
		if err != nil {
			return "", err
		}
		result = fmt.Sprintf("%s\t%s", size, path)
	}
	parts := strings.Split(result, "\t")
	return parts[0], nil
}

// formatBytesToHumanReadable takes a size in bytes as a string and converts it to a human-readable format.
func formatBytesToHumanReadable(sizeInBytes string) (string, error) {
	var size float64
	_, err := fmt.Sscanf(sizeInBytes, "%f", &size)
	if err != nil {
		return "", err
	}
	units := []string{"B", "K", "M", "G", "T", "P", "E", "Z"}
	i := 0
	for size >= 1024 {
		size /= 1024
		i++
	}
	return fmt.Sprintf("%.1f%s", size, units[i]), nil
}
