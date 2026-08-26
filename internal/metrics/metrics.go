package metrics

import (
	"sync"
)

type Stats struct {
	Allowed  int64
	Denied   int64
	Total    int64
	DenyRate float64
}

type Collector struct {
	mu      sync.Mutex
	allowed map[string]int64
	denied  map[string]int64
}

func NewCollector() *Collector {
	return &Collector{
		allowed: make(map[string]int64),
		denied:  make(map[string]int64),
	}
}

func (c *Collector) RecordAllow(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.allowed[key]++
}

func (c *Collector) RecordDeny(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.denied[key]++
}

func (c *Collector) Stats() map[string]Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	result := make(map[string]Stats)

	keys := make(map[string]struct{})
	for k := range c.allowed {
		keys[k] = struct{}{}
	}
	for k := range c.denied {
		keys[k] = struct{}{}
	}

	for key := range keys {
		allowed := c.allowed[key]
		denied := c.denied[key]
		total := allowed + denied
		var denyRate float64
		if total > 0 {
			denyRate = float64(denied) / float64(total)
		}
		result[key] = Stats{
			Allowed:  allowed,
			Denied:   denied,
			Total:    total,
			DenyRate: denyRate,
		}
	}

	return result
}

func (c *Collector) StatsFor(key string) Stats {
	c.mu.Lock()
	defer c.mu.Unlock()

	allowed := c.allowed[key]
	denied := c.denied[key]
	total := allowed + denied
	var denyRate float64
	if total > 0 {
		denyRate = float64(denied) / float64(total)
	}
	return Stats{
		Allowed:  allowed,
		Denied:   denied,
		Total:    total,
		DenyRate: denyRate,
	}
}

func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.allowed = make(map[string]int64)
	c.denied = make(map[string]int64)
}

func (c *Collector) Keys() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make(map[string]struct{})
	for k := range c.allowed {
		keys[k] = struct{}{}
	}
	for k := range c.denied {
		keys[k] = struct{}{}
	}
	result := make([]string, 0, len(keys))
	for k := range keys {
		result = append(result, k)
	}
	return result
}
