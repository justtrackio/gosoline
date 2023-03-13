package limit

import (
	"context"

	"go.uber.org/ratelimit"
)

type leakyBucketLimiter struct {
	*middlewareEmbeddable
	invocationBuilder *invocationBuilder
	lim               ratelimit.Limiter
}

func NewLeakyBucketLimiter(name string, rate int) (Limiter, error) {
	builder, err := newInvocationBuilder(name)
	if err != nil {
		return nil, err
	}

	return leakyBucketLimiter{
		middlewareEmbeddable: newMiddlewareEmbeddable(),
		invocationBuilder:    builder,
		lim:                  ratelimit.New(rate),
	}, nil
}

func (l leakyBucketLimiter) Wait(ctx context.Context, _ string) error {
	invocation := l.invocationBuilder.Build("take")

	l.middleware.OnTake(ctx, invocation)
	defer l.middleware.OnRelease(ctx, invocation)

	l.lim.Take()

	return nil
}
