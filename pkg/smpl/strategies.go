package smpl

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/tracing"
)

type (
	// Strategy is a function that determines if sampling should occur based on the context.
	// It returns isApplied (true if the strategy made a decision), isSampled (the decision), and an error.
	Strategy func(ctx context.Context) (isApplied bool, isSampled bool, err error)
)

var availableStrategies = map[string]Strategy{
	"tracing": DecideByTracing,
	"always":  DecideByAlways,
	"never":   DecideByNever,
}

// DecideByTracing makes a sampling decision based on the tracing information in the context.
// It applies if a trace is present.
func DecideByTracing(ctx context.Context) (isApplied bool, isSampled bool, err error) {
	trace := tracing.GetTraceFromContext(ctx)

	if trace == nil {
		return false, false, nil
	}

	return true, trace.Sampled, nil
}

// DecideByAlways is a strategy that always decides to sample.
func DecideByAlways(ctx context.Context) (isApplied bool, isSampled bool, err error) {
	return true, true, nil
}

// DecideByNever is a strategy that always decides not to sample.
func DecideByNever(ctx context.Context) (isApplied bool, isSampled bool, err error) {
	return true, false, nil
}
