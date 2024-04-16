package duplicator

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

const fsctlDuplicateExtentsToFile = 0x00094CF4

// duplicateExtentsToFile is a wrapper around the FSCTL_DUPLICATE_EXTENTS_TO_FILE Windows API.
// It clones the data blocks from the source file handle to the destination file handle.
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
	return windows.DeviceIoControl(
		dst,
		fsctlDuplicateExtentsToFile,
		(*byte)(unsafe.Pointer(&data)),
		uint32(unsafe.Sizeof(data)),
		nil,
		0,
		nil,
		nil,
	)
}

// CloneFile efficiently clones a file from srcFile to dstFile on Windows.
func CloneFile(srcFile, dstFile string) error {
	srcHandle, err := windows.CreateFile(windows.StringToUTF16Ptr(srcFile),
		windows.GENERIC_READ, 0, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
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

	return duplicateExtentsToFile(dstHandle, srcHandle, srcFileSize)
}
