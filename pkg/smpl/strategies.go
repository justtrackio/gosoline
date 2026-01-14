package smpl

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/tracing"
)

type (
	// Strategy is a function that determines if sampling should occur based on the context.
	// It returns isApplied (true if the strategy made a decision), isSampled (the decision), and an error.
	Strategy        func(ctx context.Context) (isApplied bool, isSampled bool, err error)
	StrategyFactory func(ctx context.Context, config cfg.Config) (Strategy, error)
)

var availableStrategies = map[string]StrategyFactory{
	"tracing":       DecideByTracing,
	"always":        DecideByAlways,
	"never":         DecideByNever,
	"probabilistic": DecideByProbabilistic,
}

// AddStrategy adds a new named strategy factory to the available strategies.
func AddStrategy(name string, strategy StrategyFactory) {
	availableStrategies[name] = strategy
}

// DecideByTracing creates a strategy that delegates to the tracing context.
// If a trace exists in the context, its sampling decision is used.
func DecideByTracing(ctx context.Context, config cfg.Config) (Strategy, error) {
	return func(ctx context.Context) (isApplied bool, isSampled bool, err error) {
		trace := tracing.GetTraceFromContext(ctx)

		if trace == nil {
			return false, false, nil
		}

		return true, trace.Sampled, nil
	}, nil
}

// DecideByAlways creates a strategy that always samples.
func DecideByAlways(ctx context.Context, config cfg.Config) (Strategy, error) {
	return func(ctx context.Context) (isApplied bool, isSampled bool, err error) {
		return true, true, nil
	}, nil
}

// DecideByNever creates a strategy that never samples.
func DecideByNever(ctx context.Context, config cfg.Config) (Strategy, error) {
	return func(ctx context.Context) (isApplied bool, isSampled bool, err error) {
		return true, false, nil
	}, nil
}
