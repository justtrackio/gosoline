package smpl

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
)

type ProbabilisticSettings struct {
	Interval            time.Duration `cfg:"interval" default:"1s"`
	FixedSampleCount    int           `cfg:"fixed_sample_count" default:"1"`
	ExtraRatePercentage int           `cfg:"extra_rate_percentage" default:"5"`
}

// probabilisticStrategy holds the state for probabilistic sampling.
// It guarantees a configurable number of sampled=true decisions per interval (when invoked),
// plus an additional configurable percentage probability for subsequent calls in that interval.
type probabilisticStrategy struct {
	mu                       sync.Mutex
	clock                    clock.Clock
	settings                 *ProbabilisticSettings
	randFloat64              func() float64
	currentInterval          int64
	fixedSamplesThisInterval int
}

// DecideByProbabilistic creates a new probabilistic sampling strategy.
// It guarantees a configurable number of sampled=true decisions per interval (when invoked),
// plus an additional configurable percentage probability for subsequent calls in that interval.
func DecideByProbabilistic(ctx context.Context, config cfg.Config) (Strategy, error) {
	settings := &ProbabilisticSettings{}
	if err := config.UnmarshalKey("sampling.settings.probabilistic", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings.probabilistic: %w", err)
	}

	s := &probabilisticStrategy{
		clock:       clock.NewRealClock(),
		settings:    settings,
		randFloat64: rand.Float64,
	}

	return func(ctx context.Context) (isApplied bool, isSampled bool, err error) {
		return s.decide(ctx)
	}, nil
}

// NewProbabilisticStrategyWithInterfaces creates a probabilistic strategy with injected clock, RNG, and settings for testing.
func NewProbabilisticStrategyWithInterfaces(clk clock.Clock, settings *ProbabilisticSettings, randFloat64 func() float64) Strategy {
	s := &probabilisticStrategy{
		clock:       clk,
		settings:    settings,
		randFloat64: randFloat64,
	}

	return func(ctx context.Context) (isApplied bool, isSampled bool, err error) {
		return s.decide(ctx)
	}
}

// decide implements the probabilistic sampling logic.
// It always applies (isApplied=true) and never returns an error.
func (s *probabilisticStrategy) decide(ctx context.Context) (isApplied bool, isSampled bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Calculate current interval bucket based on settings.Interval
	nowNanos := s.clock.Now().UnixNano()
	intervalNanos := s.settings.Interval.Nanoseconds()
	currentInterval := nowNanos / intervalNanos

	// Reset counters on interval boundary
	if currentInterval != s.currentInterval {
		s.currentInterval = currentInterval
		s.fixedSamplesThisInterval = 0
	}

	// Guarantee fixed sample count per interval
	if s.fixedSamplesThisInterval < s.settings.FixedSampleCount {
		s.fixedSamplesThisInterval++

		return true, true, nil
	}

	// Additional probability for subsequent calls
	extraRate := float64(s.settings.ExtraRatePercentage) / 100.0
	sampled := s.randFloat64() < extraRate

	return true, sampled, nil
}
