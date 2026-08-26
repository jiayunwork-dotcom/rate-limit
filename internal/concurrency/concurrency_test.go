package concurrency

import (
	"sync"
	"testing"
)

func TestSemaphoreTryAcquire(t *testing.T) {
	s := NewSemaphore(3)

	for i := 0; i < 3; i++ {
		if !s.TryAcquire() {
			t.Errorf("TryAcquire %d should succeed", i+1)
		}
	}

	if s.TryAcquire() {
		t.Error("TryAcquire should fail when all slots taken")
	}

	if avail := s.Available(); avail != 0 {
		t.Errorf("available = %d, want 0", avail)
	}
}

func TestSemaphoreRelease(t *testing.T) {
	s := NewSemaphore(2)

	s.TryAcquire()
	s.TryAcquire()

	if s.TryAcquire() {
		t.Error("should not acquire when full")
	}

	s.Release()

	if !s.TryAcquire() {
		t.Error("should acquire after release")
	}
}

func TestSemaphoreAcquireBlocking(t *testing.T) {
	s := NewSemaphore(1)
	s.TryAcquire()

	var wg sync.WaitGroup
	wg.Add(1)

	acquired := make(chan bool, 1)
	go func() {
		defer wg.Done()
		result := s.Acquire()
		acquired <- result
	}()

	s.Release()
	wg.Wait()

	if result := <-acquired; !result {
		t.Error("blocked Acquire should eventually succeed")
	}
}

func TestSemaphoreAvailable(t *testing.T) {
	s := NewSemaphore(5)

	if avail := s.Available(); avail != 5 {
		t.Errorf("initial available = %d, want 5", avail)
	}

	s.TryAcquire()
	s.TryAcquire()

	if avail := s.Available(); avail != 3 {
		t.Errorf("available after 2 acquires = %d, want 3", avail)
	}

	if inUse := s.InUse(); inUse != 2 {
		t.Errorf("in use = %d, want 2", inUse)
	}
}

func TestSemaphoreLimit(t *testing.T) {
	s := NewSemaphore(10)
	if l := s.Limit(); l != 10 {
		t.Errorf("limit = %d, want 10", l)
	}
}

func TestSemaphoreInvalidLimit(t *testing.T) {
	s := NewSemaphore(0)
	if l := s.Limit(); l != 1 {
		t.Errorf("limit with 0 input = %d, want 1", l)
	}

	s = NewSemaphore(-5)
	if l := s.Limit(); l != 1 {
		t.Errorf("limit with -5 input = %d, want 1", l)
	}
}
