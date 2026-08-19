// Package clock provides the time source abstraction used by the limiters.
package clock

import "time"

// Clock returns the current time.
type Clock interface {
	Now() time.Time
}

// Real is the wall clock.
type Real struct{}

// Now returns the real current time.
func (Real) Now() time.Time { return time.Now() }

// Fake is a manually advanced clock for deterministic tests and scenario
// replay.
type Fake struct {
	Current time.Time
}

// Now returns the fake current time.
func (f *Fake) Now() time.Time { return f.Current }

// Advance moves the fake clock forward by d (d may be negative).
func (f *Fake) Advance(d time.Duration) { f.Current = f.Current.Add(d) }
