package clock

import "time"

// Clock provides time-related operations (can be faked in tests).
type Clock interface {
	Now() time.Time
}

// RealClock uses the system clock.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now() }
