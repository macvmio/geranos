package sparsefile

import "bytes"

var zeroBuf = make([]byte, maxBufSize)

func isAllZeroes(p []byte) bool {
	// bytes.Equal is optimized version, 10x faster than simple loop
	return bytes.Equal(p, zeroBuf[:len(p)])
}
