// Package concurrency implements a concurrency limiter using a semaphore pattern.
//
// It limits the number of concurrent operations that can be in-flight
// at any given time, preventing resource exhaustion.
package concurrency

import (
	"sync"
	"sync/atomic"
)

// Semaphore implements a counting semaphore for concurrency limiting.
type Semaphore struct {
	limit   int
	current atomic.Int64
	mu      sync.Mutex
	waiters []chan struct{}
}

// NewSemaphore creates a new semaphore with the given concurrency limit.
func NewSemaphore(limit int) *Semaphore {
	if limit <= 0 {
		limit = 1
	}
	return &Semaphore{
		limit:   limit,
		waiters: make([]chan struct{}, 0),
	}
}

// Acquire blocks until a slot is available, then acquires it.
// Returns true when the slot is acquired.
func (s *Semaphore) Acquire() bool {
	if s.TryAcquire() {
		return true
	}

	// 创建等待通道
	ch := make(chan struct{})
	s.mu.Lock()
	s.waiters = append(s.waiters, ch)
	s.mu.Unlock()

	// 等待通知
	<-ch
	return true
}

// TryAcquire attempts to acquire a slot without blocking.
// Returns true if successful, false if no slots are available.
func (s *Semaphore) TryAcquire() bool {
	for {
		current := s.current.Load()
		if current >= int64(s.limit) {
			return false
		}
		if s.current.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

// Release releases a previously acquired slot.
// If there are waiters, one will be notified.
func (s *Semaphore) Release() {
	s.mu.Lock()

	if len(s.waiters) > 0 {
		// 唤醒第一个等待者
		ch := s.waiters[0]
		s.waiters = s.waiters[1:]
		s.mu.Unlock()

		// 通知令牌传递给等待者，不需要递减
		close(ch)
		return
	}

	s.mu.Unlock()
	s.current.Add(-1)
}

// Available returns the number of available slots.
func (s *Semaphore) Available() int {
	current := s.current.Load()
	available := int64(s.limit) - current
	if available < 0 {
		return 0
	}
	return int(available)
}

// Limit returns the maximum concurrency limit.
func (s *Semaphore) Limit() int {
	return s.limit
}

// InUse returns the number of currently acquired slots.
func (s *Semaphore) InUse() int {
	return int(s.current.Load())
}

// Waiting returns the number of goroutines waiting to acquire.
func (s *Semaphore) Waiting() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.waiters)
}
