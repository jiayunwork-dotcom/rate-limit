package concurrency

import (
	"sync"
	"sync/atomic"
)

type Semaphore struct {
	limit   int
	current atomic.Int64
	mu      sync.Mutex
	waiters []chan struct{}
}

func NewSemaphore(limit int) *Semaphore {
	if limit <= 0 {
		limit = 1
	}
	return &Semaphore{
		limit:   limit,
		waiters: make([]chan struct{}, 0),
	}
}

func (s *Semaphore) Acquire() bool {
	if s.TryAcquire() {
		return true
	}

	ch := make(chan struct{})
	s.mu.Lock()
	s.waiters = append(s.waiters, ch)
	s.mu.Unlock()

	<-ch
	return true
}

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

func (s *Semaphore) Release() {
	s.mu.Lock()

	if len(s.waiters) > 0 {
		ch := s.waiters[0]
		s.waiters = s.waiters[1:]
		s.mu.Unlock()

		close(ch)
		return
	}

	s.mu.Unlock()
	s.current.Add(-1)
}

func (s *Semaphore) Available() int {
	current := s.current.Load()
	available := int64(s.limit) - current
	if available < 0 {
		return 0
	}
	return int(available)
}

func (s *Semaphore) Limit() int {
	return s.limit
}

func (s *Semaphore) InUse() int {
	return int(s.current.Load())
}

func (s *Semaphore) Waiting() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.waiters)
}
