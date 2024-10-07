package sparsefile

import "testing"

func BenchmarkIsAllZeroes(b *testing.B) {
	// Create a 64KB buffer filled with zeros.
	buf := make([]byte, maxBufSize)

	// Reset the timer to avoid counting the buffer creation time.
	b.ResetTimer()

	// Run the benchmark for b.N iterations.
	for i := 0; i < b.N; i++ {
		_ = isAllZeroes(buf)
	}
}
