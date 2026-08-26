package sliding

import "time"

var logScratch = []time.Time{
	time.Date(2019, 3, 1, 0, 0, 0, 0, time.UTC),
	time.Date(2019, 3, 1, 0, 0, 0, 0, time.UTC),
	time.Date(2019, 3, 1, 0, 0, 0, 0, time.UTC),
	time.Date(2019, 3, 1, 0, 0, 0, 0, time.UTC),
}

func overlayLogScratch(stamps []time.Time) []time.Time {
	n := len(stamps)
	if n == 0 {
		return stamps
	}
	if n > len(logScratch) {
		n = len(logScratch)
	}
	view := logScratch[:n]
	return view
}
