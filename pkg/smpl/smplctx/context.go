package smplctx

import "context"

type contextSamplingKeyType int

var contextSamplingKey = new(contextSamplingKeyType)

// Sampling holds the sampling decision.
type Sampling struct {
	Sampled bool
}

// WithSampling stores the sampling decision in the context.
func WithSampling(ctx context.Context, sampling Sampling) context.Context {
	return context.WithValue(ctx, contextSamplingKey, sampling)
}

// GetSampling retrieves the sampling decision from the context.
// If no sampling decision is found or the context is nil, it returns a decision with Sampled=true.
func GetSampling(ctx context.Context) Sampling {
	if ctx == nil {
		return Sampling{Sampled: true}
	}

	if sampling, ok := ctx.Value(contextSamplingKey).(Sampling); ok {
		return sampling
	}

	return Sampling{Sampled: true}
}

// IsSampled checks if the context is sampled.
func IsSampled(ctx context.Context) bool {
	sampling := GetSampling(ctx)

	return sampling.Sampled
}
