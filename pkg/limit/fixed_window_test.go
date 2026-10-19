package limit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	clockMocks "github.com/justtrackio/gosoline/pkg/clock/mocks"
	limitMocks "github.com/justtrackio/gosoline/pkg/limit/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/require"
)

func TestFixedWindowWait(t *testing.T) {
	backendErr := errors.New("backend unavailable")
	type increment struct {
		count int
		ttl   time.Duration
		err   error
	}

	tests := []struct {
		name       string
		increments []increment
		waits      []time.Duration
		wantErr    error
	}{
		{
			name:       "available capacity",
			increments: []increment{{count: 2}},
		},
		{
			name:       "waiting request is counted in the next window",
			increments: []increment{{count: 3, ttl: 10 * time.Second}, {count: 1}},
			waits:      []time.Duration{10 * time.Second},
		},
		{
			name: "new requests fill the next window before the waiting request",
			increments: []increment{
				{count: 3, ttl: 10 * time.Second},
				{count: 3, ttl: 10 * time.Second},
				{count: 1},
			},
			waits: []time.Duration{10 * time.Second, 10 * time.Second},
		},
		{
			name:       "queued requests skip full windows and still acquire capacity",
			increments: []increment{{count: 5, ttl: 10 * time.Second}, {count: 1}},
			waits:      []time.Duration{20 * time.Second},
		},
		{
			name:       "zero TTL does not cause a busy loop",
			increments: []increment{{count: 3}, {count: 1}},
			waits:      []time.Duration{time.Second},
		},
		{
			name:       "backend error",
			increments: []increment{{err: backendErr}},
			wantErr:    backendErr,
		},
		{
			name:       "backend error after waiting",
			increments: []increment{{count: 3, ttl: 10 * time.Second}, {err: backendErr}},
			waits:      []time.Duration{10 * time.Second},
			wantErr:    backendErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()
			backend := limitMocks.NewIncrementer(t)
			for _, result := range tt.increments {
				backend.EXPECT().Increment(matcher.Context, "partner").
					Return(&result.count, &result.ttl, result.err).Once()
			}
			fakeClock := clockMocks.NewClock(t)
			for _, wait := range tt.waits {
				timer := clockMocks.NewTimer(t)
				fired := make(chan time.Time, 1)
				fired <- time.Time{}
				timer.EXPECT().Chan().Return(fired).Once()
				fakeClock.EXPECT().NewTimer(wait).Return(timer).Once()
			}

			limiter := newTestFixedWindow(t, backend, fakeClock)
			err := limiter.Wait(ctx, "partner")

			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestFixedWindowWaitCancellation(t *testing.T) {
	for _, alreadyCanceled := range []bool{false, true} {
		name := "while waiting"
		if alreadyCanceled {
			name = "before acquiring capacity"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			backend := limitMocks.NewIncrementer(t)
			fakeClock := clockMocks.NewClock(t)
			if alreadyCanceled {
				cancel()
			} else {
				count, ttl := 3, 10*time.Second
				backend.EXPECT().Increment(matcher.Context, "partner").Return(&count, &ttl, nil).Once()
				timer := clockMocks.NewTimer(t)
				timer.EXPECT().Chan().Return(nil).Once()
				timer.EXPECT().Stop().Return(true).Once()
				fakeClock.EXPECT().NewTimer(10 * time.Second).Run(func(time.Duration) {
					cancel()
				}).Return(timer).Once()
			}

			limiter := newTestFixedWindow(t, backend, fakeClock)
			require.ErrorIs(t, limiter.Wait(ctx, "partner"), context.Canceled)
		})
	}
}

func newTestFixedWindow(t *testing.T, backend Incrementer, c clock.Clock) *fixedWindow {
	t.Helper()
	builder, err := newInvocationBuilder("test")
	require.NoError(t, err)

	return NewFixedWindowLimiter(backend, c, FixedWindowConfig{
		Name:   "test",
		Cap:    2,
		Window: 10 * time.Second,
	}, builder)
}
