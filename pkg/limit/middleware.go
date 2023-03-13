package limit

import "golang.org/x/net/context"

type Middleware interface {
	OnTake(ctx context.Context, i Invocation)
	OnRelease(ctx context.Context, i Invocation)
	OnError(ctx context.Context, i Invocation)
	OnThrottle(ctx context.Context, i Invocation)
}

type MiddlewareFactory func() Middleware

type chainedMiddleware struct {
	m []Middleware
}

func (c chainedMiddleware) OnTake(ctx context.Context, i Invocation) {
	for _, m := range c.m {
		m.OnTake(ctx, i)
	}
}

func (c chainedMiddleware) OnRelease(ctx context.Context, i Invocation) {
	for _, m := range c.m {
		m.OnRelease(ctx, i)
	}
}

func (c chainedMiddleware) OnError(ctx context.Context, i Invocation) {
	for _, m := range c.m {
		m.OnError(ctx, i)
	}
}

func (c chainedMiddleware) OnThrottle(ctx context.Context, i Invocation) {
	for _, m := range c.m {
		m.OnThrottle(ctx, i)
	}
}

func ChainMiddleware(fs ...MiddlewareFactory) Middleware {
	var m []Middleware
	for _, f := range fs {
		m = append(m, f())
	}

	return chainedMiddleware{m}
}
