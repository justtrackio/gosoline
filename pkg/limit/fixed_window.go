package limit

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
)

type Incrementer interface {
	Increment(ctx context.Context, prefix string) (incr *int, ttl *time.Duration, err error)
}

type fixedWindow struct {
	*middlewareEmbeddable
	backend           Incrementer
	clock             clock.Clock
	config            FixedWindowConfig
	invocationBuilder *invocationBuilder
}

func NewFixedWindowLimiter(backend Incrementer, clock clock.Clock, config FixedWindowConfig, builder *invocationBuilder) *fixedWindow {
	return &fixedWindow{
		middlewareEmbeddable: newMiddlewareEmbeddable(),
		clock:                clock,
		backend:              backend,
		config:               config,
		invocationBuilder:    builder,
	}
}

func (f fixedWindow) Wait(ctx context.Context, prefix string) (err error) {
	invocation := f.invocationBuilder.Build(prefix)

	f.middleware.OnTake(ctx, invocation)
	defer func() {
		if err != nil {
			f.middleware.OnError(ctx, invocation)
		} else {
			f.middleware.OnRelease(ctx, invocation)
		}
	}()

	incr, ttl, err := f.backend.Increment(ctx, prefix)
	if err != nil {
		return err
	}

	// The incrementer will always start at 1, but it is nicer to calculate the
	// waiting time with the counter value starting at 0
	increment := *incr - 1

	capacity := f.config.Cap
	if increment >= capacity {
		f.middleware.OnThrottle(ctx, invocation)
		// If there are so many request are after exceeding the limit that we will
		// overflow the capacity again, we want those requests to wait for the next
		// window in which a requests will be possible again.
		// If the incremented value minus one is bigger or equal to our capacity,
		// we want to wait until the TTL is expired.
		// If the incremented value is bigger than 2 * increment, we want to wait
		// until the TTl is expired PLUS until the next window is open.
		queueTime := *ttl + (time.Duration((increment-capacity)/capacity) * f.config.Window)

		t := f.clock.NewTimer(queueTime)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.Chan():
				return nil
			}
		}
	}

	return nil
}
