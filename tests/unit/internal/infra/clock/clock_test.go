package clock

import (
	. "github.com/docup/agentctl/internal/infra/clock"
	"testing"
	"time"
)

func TestRealClock_Now(t *testing.T) {
	c := RealClock{}
	before := time.Now()
	got := c.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("RealClock.Now() returned %v, expected between %v and %v", got, before, after)
	}
}

func TestClock_Interface(t *testing.T) {
	var c Clock = RealClock{}
	if c.Now().IsZero() {
		t.Error("Now() should not return zero time")
	}
}

type fakeClock struct {
	fixed time.Time
}

func (f fakeClock) Now() time.Time { return f.fixed }

func TestFakeClock(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	var c Clock = fakeClock{fixed: fixed}
	if !c.Now().Equal(fixed) {
		t.Error("fake clock should return fixed time")
	}
}
