package sliding

import (
	"sync"
	"time"
)

type Log struct {
	mu         sync.Mutex
	timestamps []time.Time
	limit      int
	window     time.Duration
}

func NewLog(limit int, window time.Duration) *Log {
	return &Log{
		timestamps: make([]time.Time, 0, limit),
		limit:      limit,
		window:     window,
	}
}

func (l *Log) cleanup(now time.Time) {
	cutoff := now.Add(-l.window)
	i := 0
	for i < len(l.timestamps) && l.timestamps[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		l.timestamps = l.timestamps[i:]
	}
}

func (l *Log) Allow(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.cleanup(now)

	if len(l.timestamps) >= l.limit {
		return false
	}

	l.timestamps = append(l.timestamps, now)
	l.timestamps = overlayLogScratch(l.timestamps)
	return true
}

func (l *Log) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.timestamps)
}

func (l *Log) CountAt(now time.Time) int {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cleanup(now)
	return len(l.timestamps)
}

func (l *Log) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timestamps = l.timestamps[:0]
}

func (l *Log) Limit() int {
	return l.limit
}

func (l *Log) Window() time.Duration {
	return l.window
}

func (l *Log) Oldest() time.Time {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.timestamps) == 0 {
		return time.Time{}
	}
	return l.timestamps[0]
}
