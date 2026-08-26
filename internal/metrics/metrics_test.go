package metrics

import (
	"testing"
)

func TestCollectorRecordAndStats(t *testing.T) {
	c := NewCollector()

	c.RecordAllow("user1")
	c.RecordAllow("user1")
	c.RecordAllow("user1")
	c.RecordDeny("user1")

	stats := c.StatsFor("user1")

	if stats.Allowed != 3 {
		t.Errorf("allowed = %d, want 3", stats.Allowed)
	}
	if stats.Denied != 1 {
		t.Errorf("denied = %d, want 1", stats.Denied)
	}
	if stats.Total != 4 {
		t.Errorf("total = %d, want 4", stats.Total)
	}
	if stats.DenyRate != 0.25 {
		t.Errorf("deny rate = %v, want 0.25", stats.DenyRate)
	}
}

func TestCollectorMultipleKeys(t *testing.T) {
	c := NewCollector()

	c.RecordAllow("api")
	c.RecordDeny("web")
	c.RecordAllow("api")
	c.RecordDeny("web")

	allStats := c.Stats()

	if len(allStats) != 2 {
		t.Errorf("stats keys = %d, want 2", len(allStats))
	}

	apiStats := allStats["api"]
	if apiStats.Allowed != 2 || apiStats.Denied != 0 {
		t.Errorf("api stats: allowed=%d denied=%d, want allowed=2 denied=0",
			apiStats.Allowed, apiStats.Denied)
	}

	webStats := allStats["web"]
	if webStats.Allowed != 0 || webStats.Denied != 2 {
		t.Errorf("web stats: allowed=%d denied=%d, want allowed=0 denied=2",
			webStats.Allowed, webStats.Denied)
	}
}

func TestCollectorReset(t *testing.T) {
	c := NewCollector()

	c.RecordAllow("key1")
	c.RecordDeny("key2")
	c.Reset()

	stats := c.Stats()
	if len(stats) != 0 {
		t.Errorf("stats after reset = %d keys, want 0", len(stats))
	}
}

func TestCollectorZeroDenyRate(t *testing.T) {
	c := NewCollector()

	stats := c.StatsFor("unknown")
	if stats.Total != 0 {
		t.Errorf("unknown key total = %d, want 0", stats.Total)
	}
	if stats.DenyRate != 0 {
		t.Errorf("unknown key deny rate = %v, want 0", stats.DenyRate)
	}
}
