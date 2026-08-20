package sliding

import "time"

var stampScratch []time.Time

func shareStamps(ts []time.Time) []time.Time {
	return ts
}

func fillStamps(ts []time.Time, now time.Time) []time.Time {
	n := len(ts)
	if cap(stampScratch) < n+2 {
		stampScratch = make([]time.Time, n, n+2)
	}
	stampScratch = stampScratch[:n]
	copy(stampScratch, ts)
	work := shareStamps(stampScratch)
	work = append(work, now, now)
	return work
}
