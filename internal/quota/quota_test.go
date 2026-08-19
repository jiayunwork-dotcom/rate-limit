package quota

import (
	"testing"
	"time"
)

func TestManagerConsume(t *testing.T) {
	m := NewManager()
	now := time.Now()

	m.AddQuota("daily", 100, 24*time.Hour)
	m.AddQuota("hourly", 10, time.Hour)

	if !m.Consume("daily", 5, now) {
		t.Error("consume 5 from daily quota should succeed")
	}

	if r := m.Remaining("daily", now); r != 95 {
		t.Errorf("daily remaining = %d, want 95", r)
	}

	if m.Consume("hourly", 15, now) {
		t.Error("consume 15 from hourly quota (limit 10) should fail")
	}

	if m.Consume("nonexist", 1, now) {
		t.Error("consume from non-existent quota should fail")
	}
}

func TestManagerPeriodReset(t *testing.T) {
	m := NewManager()
	now := time.Now()

	m.AddQuota("hourly", 5, time.Hour)

	for i := 0; i < 5; i++ {
		m.Consume("hourly", 1, now)
	}

	if r := m.Remaining("hourly", now); r != 0 {
		t.Errorf("remaining after exhaust = %d, want 0", r)
	}

	later := now.Add(2 * time.Hour)
	if r := m.Remaining("hourly", later); r != 5 {
		t.Errorf("remaining after period = %d, want 5", r)
	}

	if !m.Consume("hourly", 3, later) {
		t.Error("consume after period reset should succeed")
	}
}

func TestManagerCheck(t *testing.T) {
	m := NewManager()
	now := time.Now()

	m.AddQuota("api", 2, time.Minute)

	if !m.Check("api", now) {
		t.Error("check should return true when quota available")
	}

	m.Consume("api", 2, now)

	if m.Check("api", now) {
		t.Error("check should return false when quota exhausted")
	}

	if m.Check("nonexist", now) {
		t.Error("check non-existent quota should return false")
	}
}

func TestManagerRemove(t *testing.T) {
	m := NewManager()
	m.AddQuota("temp", 10, time.Hour)
	m.Remove("temp")

	if r := m.Remaining("temp", time.Now()); r != 0 {
		t.Errorf("remaining after remove = %d, want 0", r)
	}
}
