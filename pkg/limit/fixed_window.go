package limit

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
)

//go:generate go run github.com/vektra/mockery/v2 --name Incrementer
type Incrementer interface {
	Increment(ctx context.Context, prefix string) (incr *int, ttl *time.Duration, err error)
}

type FixedWindowConfig struct {
	Name   string
	Cap    int
	Window time.Duration
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

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		incr, ttl, err := f.backend.Increment(ctx, prefix)
		if err != nil {
			return err
		}

		if *incr <= f.config.Cap {
			return nil
		}

		f.middleware.OnThrottle(ctx, invocation)
		// Spread queued requests across future windows according to their counter value.
		increment := *incr - 1
		capacity := f.config.Cap
		queueTime := *ttl + (time.Duration((increment-capacity)/capacity) * f.config.Window)
		// Backends with second-resolution TTLs can report zero before the window expires.
		// Avoid repeatedly incrementing the counter without waiting in that case.
		if queueTime <= 0 {
			queueTime = time.Second
		}

		t := f.clock.NewTimer(queueTime)
		select {
		case <-ctx.Done():
			t.Stop()

			return ctx.Err()
		case <-t.Chan():
			// Waiting does not reserve capacity. Recheck so this request is counted
			// in the window in which it proceeds, alongside any new requests.
		}
	}
}
