package exec_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
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
	parentCtx, cancel := context.WithCancel(s.T().Context())
	ctx, stop := exec.WithDelayedCancelContext(parentCtx, time.Minute)

	// initially the context is not canceled
	s.assertNotCanceled(ctx)

	// once we cancel it, the parent context is canceled, but not the child
	cancel()
	s.assertCanceled(parentCtx, context.Canceled)
	s.assertNotCanceled(ctx)

	// only after some time passes, the context is canceled
	s.fakeClock.BlockUntilTimers(1)
	s.fakeClock.Advance(time.Minute)
	stop() // Stop returns only after the go routine is gone
	s.assertCanceled(ctx, context.Canceled)
}

func (s *contextTestSuite) TestWithDelayedCancelContext_Stop() {
	parentCtx, cancel := context.WithCancel(s.T().Context())
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

func (s *contextTestSuite) TestWithDelayedCancelContext_StopAfterCancel() {
	parentCtx, cancel := context.WithCancel(s.T().Context())
	ctx, stop := exec.WithDelayedCancelContext(parentCtx, time.Hour)

	// initially the context is not canceled
	s.assertNotCanceled(ctx)

	// we cancel the context, so the delayed cancel will go into the waiting state
	cancel()
	// sleep some time to give the worker some time to go into the sleep mode
	time.Sleep(time.Millisecond * 100)
	s.assertCanceled(parentCtx, context.Canceled)
	s.assertNotCanceled(ctx)

	// we stop the delayed context, so it will immediately propagate the already pending cancel
	stop()
	// the context is now canceled
	<-ctx.Done()
	s.assertCanceled(ctx, context.Canceled)
}

func (s *contextTestSuite) TestWithStoppableDeadlineContext() {
	parentCtx := s.T().Context()
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
	parentCtx, cancel := context.WithCancel(s.T().Context())
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
	parentCtx, cancel := context.WithCancel(s.T().Context())
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
	parentCtx, cancelParent := context.WithCancel(s.T().Context())
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

func (s *contextTestSuite) TestConcurrentlyPrintable() {
	// if you want to know what this is: We used to use atomic.Value to store the error of the context. However, it seems
	// like this is not safe for printing - you would run into some kind of invalid memory access like this:
	//     [signal SIGSEGV: segmentation violation code=0x1 addr=0xffffffffffffffff pc=...]
	// when fmt.Sprintf accesses the atomic.Value from the context. Thus, we switched the context to use a simple mutex
	// and plain value guarded by it, which works fine. This test confirms that we can indeed print a context (which
	// happens in mocks when a mocked method is called with something like mock.AnythingOfType("*exec.stoppableContext")
	// as an argument) without crashing.
	for i := 0; i < 1000; i++ {
		ctx, cancel := exec.WithManualCancelContext(s.T().Context())
		c := make(chan struct{})
		cfn := coffin.New(s.T().Context())
		cfn.Go("cancel task", func() error {
			<-c
			cancel()

			return nil
		})
		cfn.Go("print task", func() error {
			<-c
			if s := fmt.Sprintf("%v", ctx); s == "" {
				return fmt.Errorf("should never happen")
			}

			return nil
		})
		close(c)
		s.NoError(cfn.Wait(), "Fail at iteration %d", i)
	}
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
