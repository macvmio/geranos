package layout

import (
	"fmt"
	"sync/atomic"
)

type Statistics struct {
	SourceBytesCount     atomic.Int64
	BytesWrittenCount    atomic.Int64
	BytesSkippedCount    atomic.Int64
	BytesReadCount       atomic.Int64
	BytesClonedCount     atomic.Int64
	CompressedBytesCount atomic.Int64
	MatchedSegmentsCount atomic.Int64
}

func (s *Statistics) Add(other *Statistics) {
	s.BytesWrittenCount.Add(other.BytesWrittenCount.Load())
	s.BytesSkippedCount.Add(other.BytesSkippedCount.Load())
	s.BytesReadCount.Add(other.BytesReadCount.Load())
	s.BytesClonedCount.Add(other.BytesClonedCount.Load())
	s.CompressedBytesCount.Add(other.CompressedBytesCount.Load())
	s.MatchedSegmentsCount.Add(other.MatchedSegmentsCount.Load())
	s.SourceBytesCount.Add(other.SourceBytesCount.Load())
}

func (s *Statistics) Clear() {
	s.BytesWrittenCount.Store(0)
	s.BytesSkippedCount.Store(0)
	s.BytesReadCount.Store(0)
	s.BytesClonedCount.Store(0)
	s.CompressedBytesCount.Store(0)
	s.MatchedSegmentsCount.Store(0)
}

// String formats the Statistics struct for human-readable output
func (s *Statistics) String() string {
	return fmt.Sprintf("Statistics: \n"+
		"SourceBytesCount: %d\n"+
		"BytesWrittenCount: %d\n"+
		"BytesSkippedCount: %d\n"+
		"BytesReadCount: %d\n"+
		"BytesClonedCount: %d\n"+
		"CompressedBytesCount: %d\n"+
		"MatchedSegmentsCount: %d\n",
		s.SourceBytesCount.Load(),
		s.BytesWrittenCount.Load(),
		s.BytesSkippedCount.Load(),
		s.BytesReadCount.Load(),
		s.BytesClonedCount.Load(),
		s.CompressedBytesCount.Load(),
		s.MatchedSegmentsCount.Load())
}

// ImmutableStatistics holds the immutable copy of statistics
type ImmutableStatistics struct {
	SourceBytesCount     int64
	BytesWrittenCount    int64
	BytesSkippedCount    int64
	BytesReadCount       int64
	BytesClonedCount     int64
	CompressedBytesCount int64
	MatchedSegmentsCount int64
}
