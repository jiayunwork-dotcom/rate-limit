package clock

import (
	"testing"
	"time"
)

func TestFakeClockAdvance(t *testing.T) {
	start := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	f := &Fake{Current: start}
	if !f.Now().Equal(start) {
		t.Fatalf("Now = %v, want %v", f.Now(), start)
	}
	f.Advance(90 * time.Second)
	if f.Now().Sub(start) != 90*time.Second {
		t.Errorf("advance by 90s failed, Now = %v", f.Now())
	}
	f.Advance(-30 * time.Second)
	if f.Now().Sub(start) != 60*time.Second {
		t.Errorf("negative advance failed, Now = %v", f.Now())
	}
}

func TestClockInterface(t *testing.T) {
	start := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	var c Clock = &Fake{Current: start}
	if !c.Now().Equal(start) {
		t.Errorf("fake clock Now = %v, want %v", c.Now(), start)
	}
	c = Real{}
	if c.Now().IsZero() {
		t.Error("real clock returned zero time")
	}
}
