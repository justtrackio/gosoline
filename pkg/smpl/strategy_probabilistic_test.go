package smpl_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/smpl"
	"github.com/stretchr/testify/suite"
)

type ProbabilisticStrategyTestSuite struct {
	suite.Suite
}

func TestProbabilisticStrategyTestSuite(t *testing.T) {
	suite.Run(t, new(ProbabilisticStrategyTestSuite))
}

func (s *ProbabilisticStrategyTestSuite) TestGuaranteeFirstCallPerInterval() {
	// Fixed time: always returns the same second
	fixedTime := time.Unix(1000, 0)
	fakeClock := clock.NewFakeClockAt(fixedTime)

	settings := &smpl.ProbabilisticSettings{
		Interval:            time.Second,
		FixedSampleCount:    1,
		ExtraRatePercentage: 5,
	}

	// RNG that always returns > 0.05, so probabilistic path would NOT sample
	randFunc := func() float64 { return 0.99 }

	strategy := smpl.NewProbabilisticStrategyWithInterfaces(fakeClock, settings, randFunc)

	// First call in this interval should be sampled (guarantee)
	applied, sampled, err := strategy(context.Background())
	s.True(applied, "Probabilistic strategy should always apply")
	s.True(sampled, "First call in an interval should be sampled (guarantee)")
	s.NoError(err)

	// Second call in same interval with RNG=0.99 should NOT be sampled
	applied, sampled, err = strategy(context.Background())
	s.True(applied)
	s.False(sampled, "Subsequent call with RNG > extra rate should not be sampled")
	s.NoError(err)
}

func (s *ProbabilisticStrategyTestSuite) TestMultipleFixedSamplesPerInterval() {
	fixedTime := time.Unix(1000, 0)
	fakeClock := clock.NewFakeClockAt(fixedTime)

	settings := &smpl.ProbabilisticSettings{
		Interval:            time.Second,
		FixedSampleCount:    3,
		ExtraRatePercentage: 5,
	}

	// RNG that always returns > 0.05, so probabilistic path would NOT sample
	randFunc := func() float64 { return 0.99 }

	strategy := smpl.NewProbabilisticStrategyWithInterfaces(fakeClock, settings, randFunc)

	// First 3 calls should be sampled (fixed sample count)
	for i := 0; i < 3; i++ {
		_, sampled, err := strategy(context.Background())
		s.NoError(err)
		s.True(sampled, "Call %d should be sampled (fixed sample count)", i+1)
	}

	// Fourth call should NOT be sampled (RNG=0.99 > 0.05)
	_, sampled, err := strategy(context.Background())
	s.NoError(err)
	s.False(sampled, "Call after fixed sample count should not be sampled with RNG > extra rate")
}

func (s *ProbabilisticStrategyTestSuite) TestProbabilisticPath() {
	fixedTime := time.Unix(2000, 0)
	fakeClock := clock.NewFakeClockAt(fixedTime)

	settings := &smpl.ProbabilisticSettings{
		Interval:            time.Second,
		FixedSampleCount:    1,
		ExtraRatePercentage: 5,
	}

	callCount := 0
	// First call uses guarantee, subsequent calls alternate: 0.01 (sampled), 0.99 (not sampled)
	randFunc := func() float64 {
		callCount++
		if callCount%2 == 1 {
			return 0.01 // < 0.05, should sample
		}

		return 0.99 // > 0.05, should not sample
	}

	strategy := smpl.NewProbabilisticStrategyWithInterfaces(fakeClock, settings, randFunc)

	// First call: guarantee
	_, sampled, err := strategy(context.Background())
	s.NoError(err)
	s.True(sampled, "First call should be sampled (guarantee)")

	// Second call: RNG=0.01 < 0.05 => sampled
	_, sampled, err = strategy(context.Background())
	s.NoError(err)
	s.True(sampled, "RNG < extra rate should be sampled")

	// Third call: RNG=0.99 > 0.05 => not sampled
	_, sampled, err = strategy(context.Background())
	s.NoError(err)
	s.False(sampled, "RNG > extra rate should not be sampled")

	// Fourth call: RNG=0.01 < 0.05 => sampled
	_, sampled, err = strategy(context.Background())
	s.NoError(err)
	s.True(sampled, "RNG < extra rate should be sampled")
}

func (s *ProbabilisticStrategyTestSuite) TestCustomExtraRatePercentage() {
	fixedTime := time.Unix(2000, 0)
	fakeClock := clock.NewFakeClockAt(fixedTime)

	settings := &smpl.ProbabilisticSettings{
		Interval:            time.Second,
		FixedSampleCount:    1,
		ExtraRatePercentage: 20, // 20% extra rate
	}

	// RNG returns 0.15, which is < 0.20
	randFunc := func() float64 { return 0.15 }

	strategy := smpl.NewProbabilisticStrategyWithInterfaces(fakeClock, settings, randFunc)

	// First call: guarantee
	_, sampled, err := strategy(context.Background())
	s.NoError(err)
	s.True(sampled)

	// Second call: RNG=0.15 < 0.20 => sampled
	_, sampled, err = strategy(context.Background())
	s.NoError(err)
	s.True(sampled, "RNG < 20% extra rate should be sampled")
}

func (s *ProbabilisticStrategyTestSuite) TestResetOnNewInterval() {
	fixedTime := time.Unix(3000, 0)
	fakeClock := clock.NewFakeClockAt(fixedTime)

	settings := &smpl.ProbabilisticSettings{
		Interval:            time.Second,
		FixedSampleCount:    1,
		ExtraRatePercentage: 5,
	}

	// RNG always returns > 0.05
	randFunc := func() float64 { return 0.99 }

	strategy := smpl.NewProbabilisticStrategyWithInterfaces(fakeClock, settings, randFunc)

	// First call in interval: guaranteed sampled
	_, sampled, err := strategy(context.Background())
	s.NoError(err)
	s.True(sampled, "First call in interval should be sampled")

	// Second call in same interval: not sampled (RNG=0.99)
	_, sampled, err = strategy(context.Background())
	s.NoError(err)
	s.False(sampled, "Second call in interval should not be sampled")

	// Move to next interval
	fakeClock.Advance(time.Second)

	// First call in new interval: guaranteed sampled again
	_, sampled, err = strategy(context.Background())
	s.NoError(err)
	s.True(sampled, "First call in new interval should be sampled (guarantee reset)")

	// Second call in new interval: not sampled
	_, sampled, err = strategy(context.Background())
	s.NoError(err)
	s.False(sampled, "Second call in new interval should not be sampled")
}

func (s *ProbabilisticStrategyTestSuite) TestCustomInterval() {
	fixedTime := time.Unix(0, 0)
	fakeClock := clock.NewFakeClockAt(fixedTime)

	settings := &smpl.ProbabilisticSettings{
		Interval:            5 * time.Second, // 5 second interval
		FixedSampleCount:    1,
		ExtraRatePercentage: 5,
	}

	randFunc := func() float64 { return 0.99 }

	strategy := smpl.NewProbabilisticStrategyWithInterfaces(fakeClock, settings, randFunc)

	// First call: guaranteed sampled
	_, sampled, err := strategy(context.Background())
	s.NoError(err)
	s.True(sampled)

	// Advance 3 seconds (still in same 5-second interval)
	fakeClock.Advance(3 * time.Second)

	// Second call: not sampled (RNG > 0.05, still same interval)
	_, sampled, err = strategy(context.Background())
	s.NoError(err)
	s.False(sampled, "Should not be sampled within same 5-second interval")

	// Advance 2 more seconds (now at 5 seconds, new interval)
	fakeClock.Advance(2 * time.Second)

	// Third call: guaranteed sampled (new interval)
	_, sampled, err = strategy(context.Background())
	s.NoError(err)
	s.True(sampled, "First call in new 5-second interval should be sampled")
}

func (s *ProbabilisticStrategyTestSuite) TestConcurrency() {
	fixedTime := time.Unix(4000, 0)
	fakeClock := clock.NewFakeClockAt(fixedTime)

	settings := &smpl.ProbabilisticSettings{
		Interval:            time.Second,
		FixedSampleCount:    1,
		ExtraRatePercentage: 5,
	}

	// RNG always returns > 0.05, so only the guaranteed call should be sampled
	randFunc := func() float64 { return 0.99 }

	strategy := smpl.NewProbabilisticStrategyWithInterfaces(fakeClock, settings, randFunc)

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	sampledCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, sampled, err := strategy(context.Background())
			s.NoError(err)
			if sampled {
				mu.Lock()
				sampledCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// With RNG always > 0.05, exactly 1 call should be sampled (the guarantee)
	s.Equal(1, sampledCount, "Exactly one call should be sampled (the guaranteed one)")
}

func (s *ProbabilisticStrategyTestSuite) TestConcurrencyWithMultipleFixedSamples() {
	fixedTime := time.Unix(4000, 0)
	fakeClock := clock.NewFakeClockAt(fixedTime)

	settings := &smpl.ProbabilisticSettings{
		Interval:            time.Second,
		FixedSampleCount:    5,
		ExtraRatePercentage: 0, // No extra sampling
	}

	randFunc := func() float64 { return 0.99 }

	strategy := smpl.NewProbabilisticStrategyWithInterfaces(fakeClock, settings, randFunc)

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	sampledCount := 0
	var mu sync.Mutex

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_, sampled, err := strategy(context.Background())
			s.NoError(err)
			if sampled {
				mu.Lock()
				sampledCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// With extra rate 0%, exactly 5 calls should be sampled (the fixed sample count)
	s.Equal(5, sampledCount, "Exactly 5 calls should be sampled (the fixed sample count)")
}
