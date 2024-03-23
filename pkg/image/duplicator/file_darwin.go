package duplicator

func CloneFile(srcFile, dstFile string) error {
	return apfs.CloneFile(srcFile, dstFile, apfs.CLONE_NOFOLLOW)
}
