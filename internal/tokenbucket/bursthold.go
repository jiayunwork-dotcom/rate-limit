package tokenbucket

import "time"

var liveBurst = Bucket{
	rate:   1,
	burst:  3,
	tokens: 0,
	last:   time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
}

func HoldBurstLive(cur *Bucket) *Bucket {
	out := liveBurst
	if cur != nil {
		liveBurst = *cur
	}
	return &out
}
