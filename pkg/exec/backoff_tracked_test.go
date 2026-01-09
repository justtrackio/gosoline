package exec_test

import (
	"errors"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/suite"
)

type TrackedBackOffTestSuite struct {
	suite.Suite
	fakeClock clock.FakeClock
	settings  *exec.BackoffSettings
	tracker   exec.ElapsedTimeTracker
	bo        *exec.TrackedBackOff
}

func (s *TrackedBackOffTestSuite) SetupTest() {
	s.fakeClock = clock.NewFakeClock()
	s.settings = &exec.BackoffSettings{
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		MaxElapsedTime:  5 * time.Second,
	}
	s.tracker = exec.NewErrorTriggeredElapsedTimeTrackerWithInterfaces(s.fakeClock)
	s.bo = exec.NewTrackedBackOff(s.settings, s.tracker)
}

func (s *TrackedBackOffTestSuite) TestNextBackOff_WithDefaultTracker() {
	s.tracker.Start()
	// First backoff should return an interval
	interval := s.bo.NextBackOff()
	s.NotEqual(backoff.Stop, interval)
	s.GreaterOrEqual(interval, 100*time.Millisecond)

	// Advance time past max elapsed
	s.fakeClock.Advance(6 * time.Second)

	// Now it should stop
	interval = s.bo.NextBackOff()
	s.Equal(backoff.Stop, interval)
}

func (s *TrackedBackOffTestSuite) TestNextBackOff_WithErrorTriggeredTracker() {
	s.tracker.Start()

	// Simulate blocking for 10 seconds before first error (e.g., Kafka poll)
	s.fakeClock.Advance(10 * time.Second)

	// No error yet, so elapsed should be 0 - should NOT stop
	interval := s.bo.NextBackOff()
	s.NotEqual(backoff.Stop, interval, "should not stop when no error has occurred yet")

	// Now an error occurs
	s.tracker.OnError(errors.New(s.T().Name()))

	// Should still be able to get intervals (we just started the error clock)
	interval = s.bo.NextBackOff()
	s.NotEqual(backoff.Stop, interval, "should not stop immediately after first error")

	// Advance 3 seconds (still within budget)
	s.fakeClock.Advance(3 * time.Second)
	interval = s.bo.NextBackOff()
	s.NotEqual(backoff.Stop, interval, "should not stop within max elapsed time")

	// Advance past max elapsed time from first error
	s.fakeClock.Advance(3 * time.Second) // total 6s since error
	interval = s.bo.NextBackOff()
	s.Equal(backoff.Stop, interval, "should stop after max elapsed time since first error")
}

func (s *TrackedBackOffTestSuite) TestNextBackOff_ErrorTriggeredTracker_ResetOnSuccess() {
	s.tracker.Start()

	// Error occurs
	s.tracker.OnError(errors.New(s.T().Name()))
	s.fakeClock.Advance(3 * time.Second)

	interval := s.bo.NextBackOff()
	s.NotEqual(backoff.Stop, interval)

	// Success resets the error clock
	s.tracker.OnSuccess()
	s.bo.Reset()

	// Even after 10 more seconds, we should not stop (no error active)
	s.fakeClock.Advance(10 * time.Second)
	interval = s.bo.NextBackOff()
	s.NotEqual(backoff.Stop, interval, "should not stop after success reset")

	// New error occurs
	s.tracker.OnError(errors.New(s.T().Name()))

	// New budget starts from this error
	s.fakeClock.Advance(3 * time.Second)
	interval = s.bo.NextBackOff()
	s.NotEqual(backoff.Stop, interval, "should not stop within new budget")

	s.fakeClock.Advance(3 * time.Second) // 6s since new error
	interval = s.bo.NextBackOff()
	s.Equal(backoff.Stop, interval, "should stop after exceeding new budget")
}

func (s *TrackedBackOffTestSuite) TestNextBackOff_NoMaxElapsedTime() {
	s.settings.MaxElapsedTime = 0 // disabled
	s.bo = exec.NewTrackedBackOff(s.settings, s.tracker)
	s.tracker.Start()

	// Even after a very long time, should not stop
	s.fakeClock.Advance(24 * time.Hour)

	interval := s.bo.NextBackOff()
	s.NotEqual(backoff.Stop, interval, "should never stop when MaxElapsedTime is 0")
}

func TestTrackedBackOffTestSuite(t *testing.T) {
	suite.Run(t, new(TrackedBackOffTestSuite))
}
