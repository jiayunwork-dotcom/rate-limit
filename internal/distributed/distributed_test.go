package distributed

import (
	"testing"
)

func TestClusterAllow(t *testing.T) {
	c := NewCluster(3, 10)

	allowed := 0
	for i := 0; i < 15; i++ {
		if c.Allow("user1") {
			allowed++
		}
	}

	if allowed != 10 {
		t.Errorf("allowed = %d, want 10", allowed)
	}
}

func TestClusterDifferentKeys(t *testing.T) {
	c := NewCluster(2, 5)

	for i := 0; i < 5; i++ {
		c.Allow("user1")
	}
	for i := 0; i < 5; i++ {
		c.Allow("user2")
	}

	if c.Allow("user1") {
		t.Error("user1 should be denied after limit")
	}

	if c.Allow("user2") {
		t.Error("user2 should be denied after limit")
	}
}

func TestClusterSync(t *testing.T) {
	c := NewCluster(3, 100)

	for i := 0; i < 30; i++ {
		c.Allow("key1")
	}

	c.Sync()

	if count := c.GlobalCount(); count != 30 {
		t.Errorf("global count after sync = %d, want 30", count)
	}
}

func TestClusterReset(t *testing.T) {
	c := NewCluster(2, 10)

	for i := 0; i < 10; i++ {
		c.Allow("key")
	}

	c.Reset()

	if count := c.GlobalCount(); count != 0 {
		t.Errorf("global count after reset = %d, want 0", count)
	}

	if !c.Allow("key") {
		t.Error("should allow after reset")
	}
}

func TestClusterNodeCount(t *testing.T) {
	c := NewCluster(5, 100)
	if n := c.NodeCount(); n != 5 {
		t.Errorf("node count = %d, want 5", n)
	}
}
