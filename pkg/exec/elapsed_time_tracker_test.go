package exec_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ElapsedTimeTrackerTestSuite struct {
	suite.Suite
	fakeClock clock.FakeClock
	tracker   exec.ElapsedTimeTracker
}

func TestElapsedTimeTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(ElapsedTimeTrackerTestSuite))
}

func (s *ElapsedTimeTrackerTestSuite) SetupTest() {
	s.fakeClock = clock.NewFakeClock()
	s.tracker = exec.NewDefaultElapsedTimeTrackerWithInterfaces(s.fakeClock)
}

func (s *ElapsedTimeTrackerTestSuite) TestDefaultElapsedTimeTracker_MeasuresFromStart() {
	s.tracker.Start()
	s.fakeClock.Advance(5 * time.Second)

	elapsed := s.tracker.Elapsed()
	s.Equal(5*time.Second, elapsed)
}

func (s *ElapsedTimeTrackerTestSuite) TestDefaultElapsedTimeTracker_OnErrorDoesNotAffectElapsed() {
	s.tracker.Start()
	s.fakeClock.Advance(3 * time.Second)
	s.tracker.OnError(assert.AnError)
	s.fakeClock.Advance(2 * time.Second)

	elapsed := s.tracker.Elapsed()
	s.Equal(5*time.Second, elapsed)
}

func (s *ElapsedTimeTrackerTestSuite) TestDefaultElapsedTimeTracker_OnSuccessDoesNotAffectElapsed() {
	s.tracker.Start()
	s.fakeClock.Advance(3 * time.Second)
	s.tracker.OnSuccess()
	s.fakeClock.Advance(2 * time.Second)

	elapsed := s.tracker.Elapsed()
	s.Equal(5*time.Second, elapsed)
}

func (s *ElapsedTimeTrackerTestSuite) TestErrorTriggeredElapsedTimeTracker_ReturnsZeroBeforeError() {
	s.tracker.Start()
	s.fakeClock.Advance(10 * time.Second)

	elapsed := s.tracker.Elapsed()
	s.Equal(time.Duration(0), elapsed)
}

func (s *ElapsedTimeTrackerTestSuite) TestErrorTriggeredElapsedTimeTracker_MeasuresFromFirstError() {
	s.tracker.Start()
	s.fakeClock.Advance(10 * time.Second) // Blocking time - should not count
	s.tracker.OnError(assert.AnError)
	s.fakeClock.Advance(3 * time.Second)

	elapsed := s.tracker.Elapsed()
	s.Equal(3*time.Second, elapsed)
}

func (s *ElapsedTimeTrackerTestSuite) TestErrorTriggeredElapsedTimeTracker_SecondErrorDoesNotResetTimer() {
	s.tracker.Start()
	s.tracker.OnError(assert.AnError)
	s.fakeClock.Advance(2 * time.Second)
	s.tracker.OnError(assert.AnError) // Second error should not reset
	s.fakeClock.Advance(3 * time.Second)

	elapsed := s.tracker.Elapsed()
	s.Equal(5*time.Second, elapsed)
}

func (s *ElapsedTimeTrackerTestSuite) TestErrorTriggeredElapsedTimeTracker_SuccessResetsTimer() {
	s.tracker.Start()
	s.tracker.OnError(assert.AnError)
	s.fakeClock.Advance(5 * time.Second)
	s.tracker.OnSuccess() // Should reset
	s.fakeClock.Advance(10 * time.Second)

	// After success, elapsed should be zero again
	elapsed := s.tracker.Elapsed()
	s.Equal(time.Duration(0), elapsed)
}

func (s *ElapsedTimeTrackerTestSuite) TestErrorTriggeredElapsedTimeTracker_ErrorAfterSuccessStartsFresh() {
	s.tracker.Start()
	s.tracker.OnError(assert.AnError)
	s.fakeClock.Advance(5 * time.Second)
	s.tracker.OnSuccess() // Reset
	s.fakeClock.Advance(10 * time.Second)
	s.tracker.OnError(assert.AnError) // New error
	s.fakeClock.Advance(2 * time.Second)

	elapsed := s.tracker.Elapsed()
	s.Equal(2*time.Second, elapsed)
}

func (s *ElapsedTimeTrackerTestSuite) TestErrorTriggeredElapsedTimeTracker_StartResetsState() {
	s.tracker.Start()
	s.tracker.OnError(assert.AnError)
	s.fakeClock.Advance(5 * time.Second)

	// Start again should reset everything
	s.tracker.Start()
	s.fakeClock.Advance(10 * time.Second)

	elapsed := s.tracker.Elapsed()
	s.Equal(time.Duration(0), elapsed)
}
