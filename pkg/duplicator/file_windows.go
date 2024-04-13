package duplicator

import (
	"os"
	"syscall"
	"unsafe"
)

const fsctlDuplicateExtentsToFile = 0x00094CF4

// duplicateExtentsToFile is a wrapper around the FSCTL_DUPLICATE_EXTENTS_TO_FILE Windows API.
// It clones the data blocks from the source file handle to the destination file handle.
func duplicateExtentsToFile(dst, src syscall.Handle, srcLength int64) error {
	type DuplicateExtentsData struct {
		FileHandle       syscall.Handle
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
	_, _, err := syscall.Syscall6(
		syscall.PROC_DEVICE_IO_CONTROL.Addr(),
		6,
		uintptr(dst),
		fsctlDuplicateExtentsToFile,
		uintptr(unsafe.Pointer(&data)),
		unsafe.Sizeof(data),
		0,
		0,
	)
	if err != 0 && err != syscall.Errno(0) {
		return err
	}
	return nil
}

// CloneFile efficiently clones a file from srcFile to dstFile on Windows.
func CloneFile(srcFile, dstFile string) error {
	srcHandle, err := syscall.CreateFile(&(syscall.StringToUTF16(srcFile)[0]),
		syscall.GENERIC_READ, 0, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return os.NewSyscallError("CreateFile src", err)
	}
	defer syscall.CloseHandle(srcHandle)

	dstHandle, err := syscall.CreateFile(&(syscall.StringToUTF16(dstFile)[0]),
		syscall.GENERIC_WRITE, 0, nil, syscall.CREATE_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return os.NewSyscallError("CreateFile dst", err)
	}
	defer syscall.CloseHandle(dstHandle)

	srcFileInfo, err := os.Stat(srcFile)
	if err != nil {
		return err
	}
	srcFileSize := srcFileInfo.Size()

	return duplicateExtentsToFile(dstHandle, srcHandle, srcFileSize)
}
