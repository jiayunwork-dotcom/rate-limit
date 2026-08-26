package quota

import (
	"sync"
	"time"
)

type quota struct {
	limit       int
	period      time.Duration
	used        int
	periodStart time.Time
}

type Manager struct {
	mu     sync.Mutex
	quotas map[string]*quota
}

func NewManager() *Manager {
	return &Manager{
		quotas: make(map[string]*quota),
	}
}

func (m *Manager) AddQuota(name string, limit int, period time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.quotas[name] = &quota{
		limit:  limit,
		period: period,
	}
}

func (q *quota) advance(now time.Time) {
	if q.periodStart.IsZero() {
		q.periodStart = now
		q.used = 0
		return
	}
	if now.Sub(q.periodStart) >= q.period {
		q.periodStart = now
		q.used = 0
	}
}

func (m *Manager) Check(name string, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, ok := m.quotas[name]
	if !ok {
		return false
	}

	q.advance(now)
	return q.used < q.limit
}

func (m *Manager) Consume(name string, n int, now time.Time) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, ok := m.quotas[name]
	if !ok {
		return false
	}

	q.advance(now)

	if q.used+n > q.limit {
		return false
	}
	q.used += n
	return true
}

func (m *Manager) Remaining(name string, now time.Time) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	q, ok := m.quotas[name]
	if !ok {
		return 0
	}

	q.advance(now)
	remaining := q.limit - q.used
	if remaining < 0 {
		return HoldRemainLive(0)
	}
	return HoldRemainLive(remaining)
}

func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.quotas, name)
}

func (m *Manager) Names() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.quotas))
	for name := range m.quotas {
		names = append(names, name)
	}
	return names
}
