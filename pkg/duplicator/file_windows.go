package duplicator

import (
	"errors"
	"fmt"
	"io"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const fsctlDuplicateExtentsToFile = 0x00094CF4

func CloneFileFallback(srcFile, dstFile string) error {
	fmt.Printf("CloneFileFallback: %v -> %v\n", srcFile, dstFile)
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// duplicateExtentsToFile clones data blocks from the source file handle to the destination file handle.
func duplicateExtentsToFile(dst, src windows.Handle, srcLength int64) error {
	type DuplicateExtentsData struct {
		FileHandle       windows.Handle
		SourceFileOffset int64
		TargetFileOffset int64
		ByteCount        int64
	}
	data := DuplicateExtentsData{
		FileHandle:       src,
		SourceFileOffset: 0,
		TargetFileOffset: 0,
		ByteCount:        srcLength,
	}
	return os.NewSyscallError("DeviceIoControl",
		windows.DeviceIoControl(
			dst,
			fsctlDuplicateExtentsToFile,
			(*byte)(unsafe.Pointer(&data)),
			uint32(unsafe.Sizeof(data)),
			nil,
			0,
			nil,
			nil,
		),
	)
}

// CloneFile efficiently clones a file from srcFile to dstFile on Windows.
func CloneFile(srcFile, dstFile string) error {
	srcHandle, err := windows.CreateFile(windows.StringToUTF16Ptr(srcFile),
		windows.GENERIC_READ, windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return os.NewSyscallError("CreateFile src", err)
	}
	defer windows.CloseHandle(srcHandle)

	dstHandle, err := windows.CreateFile(windows.StringToUTF16Ptr(dstFile),
		windows.GENERIC_WRITE, 0, nil, windows.CREATE_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return os.NewSyscallError("CreateFile dst", err)
	}
	defer windows.CloseHandle(dstHandle)

	srcFileInfo, err := os.Stat(srcFile)
	if err != nil {
		return err
	}
	srcFileSize := srcFileInfo.Size()

	// Attempt to clone using duplicate extents
	err = duplicateExtentsToFile(dstHandle, srcHandle, srcFileSize)
	if err != nil {
		windows.Close(srcHandle)
		windows.Close(dstHandle)
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_NOT_SUPPORTED) {
			// Fallback to traditional file copy if access is denied or operation is not supported
			return CloneFileFallback(srcFile, dstFile)
		}
		return err
	}
	return nil
}
