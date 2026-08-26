package distributed

import (
	"sync"
)

type Node struct {
	mu         sync.Mutex
	id         int
	localCount int
	localLimit int
}

type Cluster struct {
	mu          sync.Mutex
	nodes       []*Node
	globalLimit int
	globalCount int
	keyCounters map[string]int
}

func NewCluster(nodes int, globalLimit int) *Cluster {
	c := &Cluster{
		nodes:       make([]*Node, nodes),
		globalLimit: globalLimit,
		keyCounters: make(map[string]int),
	}

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

	nodeIdx := count % len(c.nodes)
	node := c.nodes[nodeIdx]
	node.mu.Lock()
	node.localCount++
	node.mu.Unlock()

	return true
}

func (c *Cluster) Sync() {
	c.mu.Lock()
	defer c.mu.Unlock()

	totalUsed := 0
	for _, node := range c.nodes {
		node.mu.Lock()
		totalUsed += node.localCount
		node.mu.Unlock()
	}

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

func (c *Cluster) NodeCount() int {
	return len(c.nodes)
}

func (c *Cluster) GlobalCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.globalCount
}

func (c *Cluster) KeyCount(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.keyCounters[key]
}
