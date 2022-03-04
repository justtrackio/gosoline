package exec_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/suite"
)

type contextTestSuite struct {
	suite.Suite
	fakeClock   clock.FakeClock
	oldProvider clock.Clock
}

func (s *contextTestSuite) SetupSuite() {
	s.oldProvider = clock.Provider
	s.fakeClock = clock.NewFakeClock()
	clock.Provider = s.fakeClock
}

func (s *contextTestSuite) TearDownSuite() {
	clock.Provider = s.oldProvider
}

func (s *contextTestSuite) TestWithDelayedCancelContext() {
	parentCtx, cancel := context.WithCancel(context.Background())
	ctx, stop := exec.WithDelayedCancelContext(parentCtx, time.Minute)

	// initially the context is not canceled
	s.assertNotCanceled(ctx)

	// once we cancel it, the parent context is canceled, but not the child
	cancel()
	s.assertCanceled(parentCtx, context.Canceled)
	s.assertNotCanceled(ctx)

	// only after some time passes, the context is canceled
	s.fakeClock.BlockUntil(1)
	s.fakeClock.Advance(time.Minute)
	stop() // Stop returns only after the go routine is gone
	s.assertCanceled(ctx, context.Canceled)
}

func (s *contextTestSuite) TestWithDelayedCancelContext_Stop() {
	parentCtx, cancel := context.WithCancel(context.Background())
	ctx, stop := exec.WithDelayedCancelContext(parentCtx, time.Minute)

	// initially the context is not canceled
	s.assertNotCanceled(ctx)

	// we stop the delayed context, so it will never propagate the cancel
	stop()
	s.assertNotCanceled(ctx)

	// once we cancel it, the parent context is canceled, but not the child
	cancel()
	s.assertCanceled(parentCtx, context.Canceled)
	// even give it some time to wrongly propagate a cancel - as this should not happen, this should not change it
	time.Sleep(time.Millisecond)
	s.assertNotCanceled(ctx)
}

func (s *contextTestSuite) TestWithStoppableDeadlineContext() {
	parentCtx := context.Background()
	ctx, stop := exec.WithStoppableDeadlineContext(parentCtx, s.fakeClock.Now().Add(time.Minute))
	defer stop()

	// initially the context is not canceled
	s.assertNotCanceled(ctx)

	// after some time passes, the context is canceled
	s.fakeClock.BlockUntilTimers(1)
	s.fakeClock.Advance(time.Minute)
	// the context is now canceled
	<-ctx.Done()
	s.assertCanceled(ctx, context.DeadlineExceeded)
}

func (s *contextTestSuite) TestWithStoppableDeadlineContext_CancelParent() {
	parentCtx, cancel := context.WithCancel(context.Background())
	ctx, stop := exec.WithStoppableDeadlineContext(parentCtx, s.fakeClock.Now().Add(time.Minute))
	defer stop()

	// initially the context is not canceled
	s.assertNotCanceled(ctx)

	// once we cancel it, the parent context is canceled, the child is also canceled
	cancel()
	// the context is now canceled
	<-ctx.Done()
	s.assertCanceled(ctx, context.Canceled)
}

func (s *contextTestSuite) TestWithStoppableDeadlineContext_Stop() {
	parentCtx, cancel := context.WithCancel(context.Background())
	ctx, stop := exec.WithStoppableDeadlineContext(parentCtx, s.fakeClock.Now().Add(time.Minute))

	// initially the context is not canceled
	s.assertNotCanceled(ctx)

	// we stop the delayed context, so it will never propagate the cancel
	stop()
	s.assertNotCanceled(ctx)

	// once we cancel it, the parent context is canceled, but not the child
	cancel()
	s.assertCanceled(parentCtx, context.Canceled)
	// even give it some time to wrongly propagate a cancel - as this should not happen, this should not change it
	time.Sleep(time.Millisecond)
	s.assertNotCanceled(ctx)
}

func (s *contextTestSuite) TestWithManualCancelContext() {
	parentCtx, cancelParent := context.WithCancel(context.Background())
	ctx, cancelChild := exec.WithManualCancelContext(parentCtx)

	// initially the context is not canceled
	s.assertNotCanceled(ctx)

	// we stop the parent context, this should not be propagated to the child
	cancelParent()
	s.assertCanceled(parentCtx, context.Canceled)
	s.assertNotCanceled(ctx)

	// once we cancel the child, it should be canceled
	cancelChild()
	s.assertCanceled(ctx, context.Canceled)
}

func (s *contextTestSuite) assertCanceled(ctx context.Context, expectedErr error) {
	select {
	case <-ctx.Done():
		s.Equal(expectedErr, ctx.Err())
	default:
		s.Fail("context was not canceled")
	}
}

func (s *contextTestSuite) assertNotCanceled(ctx context.Context) {
	select {
	case <-ctx.Done():
		s.Fail("context was canceled")
	default:
		s.NoError(ctx.Err())
	}
}

func TestContextTestSuite(t *testing.T) {
	suite.Run(t, new(contextTestSuite))
}
