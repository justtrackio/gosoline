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
	StragegyFactory func(ctx context.Context, config cfg.Config) (Strategy, error)
)

var availableStrategies = map[string]StragegyFactory{
	"tracing":       DecideByTracing,
	"always":        DecideByAlways,
	"never":         DecideByNever,
	"probabilistic": DecideByProbabilistic,
}

func AddStrategy(name string, strategy StragegyFactory) {
	availableStrategies[name] = strategy
}

func DecideByTracing(ctx context.Context, config cfg.Config) (Strategy, error) {
	return func(ctx context.Context) (isApplied bool, isSampled bool, err error) {
		trace := tracing.GetTraceFromContext(ctx)

		if trace == nil {
			return false, false, nil
		}

		return true, trace.Sampled, nil
	}, nil
}

func DecideByAlways(ctx context.Context, config cfg.Config) (Strategy, error) {
	return func(ctx context.Context) (isApplied bool, isSampled bool, err error) {
		return true, true, nil
	}, nil
}

func DecideByNever(ctx context.Context, config cfg.Config) (Strategy, error) {
	return func(ctx context.Context) (isApplied bool, isSampled bool, err error) {
		return true, false, nil
	}, nil
}
