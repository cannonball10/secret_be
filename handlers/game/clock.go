package game

import "time"

// Clock abstracts the current time so tests can advance time
// deterministically rather than sleeping.
type Clock interface {
	Now() time.Time
}

// RealClock returns time.Now().UTC().
type RealClock struct{}

// Now implements Clock.
func (RealClock) Now() time.Time { return time.Now().UTC() }

// FakeClock is a test double that returns a caller-controlled time.
type FakeClock struct {
	Current time.Time
}

// Now implements Clock.
func (f *FakeClock) Now() time.Time { return f.Current }

// Advance moves the fake clock forward by d.
func (f *FakeClock) Advance(d time.Duration) { f.Current = f.Current.Add(d) }
