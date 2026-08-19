// Package distributed implements a distributed rate limiting simulation.
//
// It simulates a cluster of nodes that coordinate rate limiting
// through periodic synchronization, without requiring external dependencies.
// Each node maintains a local counter and periodically syncs with the cluster.
package distributed

import (
	"sync"
)

// Node represents a single node in the rate limiting cluster.
type Node struct {
	mu         sync.Mutex
	id         int
	localCount int
	localLimit int
}

// Cluster manages a group of nodes for distributed rate limiting.
type Cluster struct {
	mu          sync.Mutex
	nodes       []*Node
	globalLimit int
	globalCount int
	keyCounters map[string]int
}

// NewCluster creates a new cluster with the specified number of nodes
// and a global rate limit shared across all nodes.
func NewCluster(nodes int, globalLimit int) *Cluster {
	c := &Cluster{
		nodes:       make([]*Node, nodes),
		globalLimit: globalLimit,
		keyCounters: make(map[string]int),
	}

	// 每个节点分配等份的限额
	perNode := globalLimit / nodes
	remainder := globalLimit % nodes

	for i := 0; i < nodes; i++ {
		limit := perNode
		if i < remainder {
			limit++
		}
		c.nodes[i] = &Node{
			id:         i,
			localLimit: limit,
		}
	}

	return c
}

// Allow checks if a request for the given key is allowed.
// It uses a simple round-robin approach to distribute load across nodes.
func (c *Cluster) Allow(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	count, exists := c.keyCounters[key]
	if !exists {
		count = 0
	}

	if count >= c.globalLimit {
		return false
	}

	c.keyCounters[key] = count + 1
	c.globalCount++

	// 分配到对应节点
	nodeIdx := count % len(c.nodes)
	node := c.nodes[nodeIdx]
	node.mu.Lock()
	node.localCount++
	node.mu.Unlock()

	return true
}

// Sync simulates synchronization between nodes.
// It rebalances the local limits based on actual usage.
func (c *Cluster) Sync() {
	c.mu.Lock()
	defer c.mu.Unlock()

	totalUsed := 0
	for _, node := range c.nodes {
		node.mu.Lock()
		totalUsed += node.localCount
		node.mu.Unlock()
	}

	// 重新分配剩余配额
	remaining := c.globalLimit - totalUsed
	if remaining < 0 {
		remaining = 0
	}

	perNode := remaining / len(c.nodes)
	for _, node := range c.nodes {
		node.mu.Lock()
		node.localLimit = perNode + node.localCount
		node.mu.Unlock()
	}
}

// Reset resets all counters in the cluster.
func (c *Cluster) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.globalCount = 0
	c.keyCounters = make(map[string]int)

	perNode := c.globalLimit / len(c.nodes)
	remainder := c.globalLimit % len(c.nodes)

	for i, node := range c.nodes {
		node.mu.Lock()
		node.localCount = 0
		node.localLimit = perNode
		if i < remainder {
			node.localLimit++
		}
		node.mu.Unlock()
	}
}

// NodeCount returns the number of nodes in the cluster.
func (c *Cluster) NodeCount() int {
	return len(c.nodes)
}

// GlobalCount returns the total number of allowed requests across the cluster.
func (c *Cluster) GlobalCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.globalCount
}

// KeyCount returns the number of requests for a specific key.
func (c *Cluster) KeyCount(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.keyCounters[key]
}
