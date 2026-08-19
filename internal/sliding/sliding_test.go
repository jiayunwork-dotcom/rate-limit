package sliding

import (
	"testing"
	"time"
)

func TestLogAllow(t *testing.T) {
	l := NewLog(3, time.Second)
	now := time.Now()

	for i := 0; i < 3; i++ {
		if !l.Allow(now) {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	if l.Allow(now) {
		t.Error("request should be denied when limit reached")
	}

	later := now.Add(2 * time.Second)
	if !l.Allow(later) {
		t.Error("request should be allowed after window expires")
	}
}

func TestLogCount(t *testing.T) {
	l := NewLog(10, time.Second)
	now := time.Now()

	if l.Count() != 0 {
		t.Errorf("initial count = %d, want 0", l.Count())
	}

	l.Allow(now)
	l.Allow(now.Add(100 * time.Millisecond))
	l.Allow(now.Add(200 * time.Millisecond))

	if l.Count() != 3 {
		t.Errorf("count after 3 allows = %d, want 3", l.Count())
	}

	later := now.Add(2 * time.Second)
	if count := l.CountAt(later); count != 0 {
		t.Errorf("count after window = %d, want 0", count)
	}
}

func TestLogReset(t *testing.T) {
	l := NewLog(10, time.Second)
	now := time.Now()

	l.Allow(now)
	l.Allow(now)
	l.Reset()

	if l.Count() != 0 {
		t.Errorf("count after reset = %d, want 0", l.Count())
	}
}

func TestLogOldest(t *testing.T) {
	l := NewLog(10, time.Second)

	if oldest := l.Oldest(); !oldest.IsZero() {
		t.Errorf("oldest of empty log = %v, want zero", oldest)
	}

	now := time.Now()
	l.Allow(now)
	l.Allow(now.Add(time.Millisecond))

	if oldest := l.Oldest(); !oldest.Equal(now) {
		t.Errorf("oldest = %v, want %v", oldest, now)
	}
}
