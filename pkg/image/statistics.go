package image

import "sync/atomic"

type Statistics struct {
	BytesWrittenCount    int64
	BytesSkippedCount    int64
	BytesReadCount       int64
	BytesClonedCount     int64
	CompressedBytesCount int64
	MatchedSegmentsCount int64
}

func (s *Statistics) Add(other *Statistics) {
	atomic.AddInt64(&s.BytesWrittenCount, other.BytesWrittenCount)
	atomic.AddInt64(&s.BytesSkippedCount, other.BytesSkippedCount)
	atomic.AddInt64(&s.BytesReadCount, other.BytesReadCount)
	atomic.AddInt64(&s.BytesClonedCount, other.BytesClonedCount)
	atomic.AddInt64(&s.CompressedBytesCount, other.CompressedBytesCount)
	atomic.AddInt64(&s.MatchedSegmentsCount, other.MatchedSegmentsCount)
}

func (s *Statistics) Clear() {
	atomic.StoreInt64(&s.BytesWrittenCount, 0)
	atomic.StoreInt64(&s.BytesSkippedCount, 0)
	atomic.StoreInt64(&s.BytesReadCount, 0)
	atomic.StoreInt64(&s.BytesClonedCount, 0)
	atomic.StoreInt64(&s.CompressedBytesCount, 0)
	atomic.StoreInt64(&s.MatchedSegmentsCount, 0)
}
