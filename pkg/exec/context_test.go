package exec_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/assert"
)

func TestWithDelayedCancelContext(t *testing.T) {
	oldProvider := clock.Provider
	defer func() {
		clock.Provider = oldProvider
	}()
	fakeClock := clock.NewFakeClock()
	clock.Provider = fakeClock

	parentCtx, cancel := context.WithCancel(context.Background())

	ctx := exec.WithDelayedCancelContext(parentCtx, time.Minute)

	// initially the context is not canceled
	assertNotCanceled(t, ctx)

	// once we cancel it, the parent context is canceled, but not the child
	cancel()
	assertCanceled(t, parentCtx)
	assertNotCanceled(t, ctx)

	// only after some time passes, the context is canceled
	fakeClock.BlockUntil(1)
	fakeClock.Advance(time.Minute)
	ctx.Stop() // Stop returns only after the go routine is gone
	assertCanceled(t, ctx)
}

func TestWithDelayedCancelContext_Stop(t *testing.T) {
	oldProvider := clock.Provider
	defer func() {
		clock.Provider = oldProvider
	}()
	fakeClock := clock.NewFakeClock()
	clock.Provider = fakeClock

	parentCtx, cancel := context.WithCancel(context.Background())

	ctx := exec.WithDelayedCancelContext(parentCtx, time.Minute)

	// initially the context is not canceled
	assertNotCanceled(t, ctx)

	// we stop the delayed context, so it will never propagate the cancel
	ctx.Stop()
	assertNotCanceled(t, ctx)

	// once we cancel it, the parent context is canceled, but not the child
	cancel()
	assertCanceled(t, parentCtx)
	// even give it some time to wrongly propagate a cancel - as this should not happen, this should not change it
	time.Sleep(time.Millisecond)
	assertNotCanceled(t, ctx)
}

func TestWithManualCancelContext(t *testing.T) {
	parentCtx, cancelParent := context.WithCancel(context.Background())

	ctx, cancelChild := exec.WithManualCancelContext(parentCtx)

	// initially the context is not canceled
	assertNotCanceled(t, ctx)

	// we stop the parent context, this should not be propagated to the child
	cancelParent()
	assertCanceled(t, parentCtx)
	assertNotCanceled(t, ctx)

	// once we cancel the child, it should be canceled
	cancelChild()
	assertCanceled(t, ctx)
}

func assertCanceled(t *testing.T, ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
		assert.Fail(t, "context was not canceled")
	}
}

func assertNotCanceled(t *testing.T, ctx context.Context) {
	select {
	case <-ctx.Done():
		assert.Fail(t, "context was canceled")
	default:
		return
	}
}
