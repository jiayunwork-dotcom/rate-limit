// Package quota implements quota management for rate limiting.
//
// It supports managing multiple named quotas with different limits
// and periods (e.g., daily limits, hourly limits).
package quota

import (
	"sync"
	"time"
)

// quota represents a single named quota with its configuration and state.
type quota struct {
	limit       int
	period      time.Duration
	used        int
	periodStart time.Time
}

// Manager manages multiple named quotas.
type Manager struct {
	mu     sync.Mutex
	quotas map[string]*quota
}

// NewManager creates a new quota manager.
func NewManager() *Manager {
	return &Manager{
		quotas: make(map[string]*quota),
	}
}

// AddQuota registers a new named quota with the specified limit and period.
// If a quota with the same name already exists, it will be overwritten.
func (m *Manager) AddQuota(name string, limit int, period time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.quotas[name] = &quota{
		limit:  limit,
		period: period,
	}
}

// advance resets the quota if the current period has expired.
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

// Check checks if the named quota has remaining capacity without consuming it.
// Returns false if the quota does not exist or is exhausted.
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

// Consume attempts to consume n units from the named quota.
// Returns true if successful, false if quota does not exist or insufficient capacity.
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

// Remaining returns the remaining capacity for the named quota.
// Returns 0 if the quota does not exist.
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
		return 0
	}
	return remaining
}

// Remove removes a named quota from the manager.
func (m *Manager) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.quotas, name)
}

// Names returns all registered quota names.
func (m *Manager) Names() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.quotas))
	for name := range m.quotas {
		names = append(names, name)
	}
	return names
}
