package models

import (
	"time"

	"github.com/spf13/viper"
)

// Bucket holds the epoch and bucket size.
// It calculates bucket number from timestamps and bucket
// ranges.
type Bucket struct {
	start      time.Time
	bucketSize float64
}

// Bucket constants
// Hardcoded since we do not want to make it so easy to change.
// If you really want to change after being in production, you need to
// sync with the message writer the new bucket size, maybe use an if
// statement to start both together in the same date and, after
// messages are deleted due to ttl, you remove the if statement and use
// only the latest bucket size.
var (
	StartDate, _ = time.Parse("01-02-2006", "01-01-2018")
	Day          = 24 * time.Hour
	BucketSize   = 3 * Day / time.Second
)

// NewBucket returns an instance of bucket
func NewBucket(config *viper.Viper) *Bucket {
	return &Bucket{
		start:      StartDate,
		bucketSize: float64(BucketSize),
	}
}

// Get returns the number of buckets (periods of time)
// since start
func (b *Bucket) Get(from int64) int {
	diff := time.Unix(from, 0).Sub(b.start).Seconds()

	buckets := int(diff / b.bucketSize)
	if buckets < 0 {
		buckets = 0
	}

	return buckets
}

// Range returns a list of buckets starting in from and ending in since
func (b *Bucket) Range(from, to int64) []int {
	bucketFrom := b.Get(from)
	bucketTo := b.Get(to)

	buckets := make([]int, bucketFrom-bucketTo+1)
	idx := 0
	for i := bucketTo; i <= bucketFrom; i++ {
		buckets[idx] = i
		idx = idx + 1
	}

	return buckets
}

// GetBuckets returns the buckets starting with from until
// have qnt buckets
func (b *Bucket) GetBuckets(from int64, qnt int) []int {
	current := b.Get(from)
	buckets := []int{}
	for i := 0; i < qnt && current-i > 0; i++ {
		buckets = append(buckets, current-i)
	}
	return buckets
}
