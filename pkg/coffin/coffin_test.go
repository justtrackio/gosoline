package coffin_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/stretchr/testify/suite"
)

type coffinTestSuite struct {
	suite.Suite
	cfn coffin.Coffin
}

func TestCoffin(t *testing.T) {
	suite.Run(t, new(coffinTestSuite))
}

func (s *coffinTestSuite) SetupTest() {
	s.cfn = coffin.New(s.T().Context(), coffin.WithLabels(map[string]string{
		"test": s.T().Name(),
	}))
}

func (s *coffinTestSuite) TestGo() {
	s.cfn.GoWithContext("test", func(context.Context) error {
		s.Equal(1, s.cfn.Running())
		s.Equal(1, s.cfn.Started())
		s.Equal(0, s.cfn.Terminated())
		err := s.cfn.Err()
		s.NoError(err)

		return nil
	})

	err := s.cfn.Wait()
	s.NoError(err)
	s.Equal(0, s.cfn.Running())
	s.Equal(1, s.cfn.Started())
	s.Equal(1, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestGoWithError() {
	s.cfn.Go("test", func() error {
		s.Equal(1, s.cfn.Running())
		s.Equal(1, s.cfn.Started())
		s.Equal(0, s.cfn.Terminated())
		err := s.cfn.Err()
		s.NoError(err)

		return fmt.Errorf("test error")
	})

	err := s.cfn.Wait()
	s.EqualError(err, `failed to execute task "test" from package "coffin_test": test error`)
	s.Equal(0, s.cfn.Running())
	s.Equal(1, s.cfn.Started())
	s.Equal(1, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestGoWithPanic() {
	s.cfn.Go("test", func() error {
		s.Equal(1, s.cfn.Running())
		s.Equal(1, s.cfn.Started())
		s.Equal(0, s.cfn.Terminated())
		err := s.cfn.Err()
		s.NoError(err)

		panic("test panic")
	})

	err := s.cfn.Wait()
	s.Error(err)
	s.Contains(err.Error(), "test panic")
	s.Equal(0, s.cfn.Running())
	s.Equal(1, s.cfn.Started())
	s.Equal(1, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestGoWithContext() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.cfn.GoWithContext("test", func(ctxArg context.Context) error {
		s.Equal(ctx, ctxArg)

		s.Equal(1, s.cfn.Running())
		s.Equal(1, s.cfn.Started())
		s.Equal(0, s.cfn.Terminated())
		err := s.cfn.Err()
		s.NoError(err)

		return nil
	}, coffin.WithContext(ctx))

	err := s.cfn.Wait()
	s.NoError(err)
	s.Equal(0, s.cfn.Running())
	s.Equal(1, s.cfn.Started())
	s.Equal(1, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestConcurrentGo() {
	for i := 0; i < 100; i++ {
		s.cfn.Go(fmt.Sprintf("test-%02d", i), func() error {
			return nil
		}, coffin.WithLabels(map[string]string{
			"index": strconv.Itoa(i),
		}))
	}

	err := s.cfn.Wait()
	s.NoError(err)
	s.Equal(0, s.cfn.Running())
	s.Equal(100, s.cfn.Started())
	s.Equal(100, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestMultipleErrors() {
	s.cfn.Go("task1", func() error {
		return fmt.Errorf("error 1")
	})
	s.cfn.Go("task2", func() error {
		return fmt.Errorf("error 2")
	})

	err := s.cfn.Wait()
	s.Error(err)
	s.Contains(err.Error(), "error 1")
	s.Contains(err.Error(), "error 2")
	s.Equal(0, s.cfn.Running())
	s.Equal(2, s.cfn.Started())
	s.Equal(2, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestMixedSuccessFailure() {
	s.cfn.Go("success-task", func() error {
		return nil
	})
	s.cfn.Go("failure-task", func() error {
		return fmt.Errorf("task failed")
	})

	err := s.cfn.Wait()
	s.EqualError(err, `failed to execute task "failure-task" from package "coffin_test": task failed`)
	s.Equal(0, s.cfn.Running())
	s.Equal(2, s.cfn.Started())
	s.Equal(2, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestNestedGo() {
	s.cfn.Go("parent", func() error {
		s.cfn.Go("child", func() error {
			return nil
		})

		return nil
	})

	err := s.cfn.Wait()
	s.NoError(err)
	s.Equal(0, s.cfn.Running())
	s.Equal(2, s.cfn.Started())
	s.Equal(2, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestCoffinReuse() {
	for i := 1; i <= 100; i++ {
		s.cfn.Go("task", func() error {
			return nil
		})

		err := s.cfn.Wait()
		s.NoError(err)
		s.Equal(0, s.cfn.Running())
		s.Equal(i, s.cfn.Started())
		s.Equal(i, s.cfn.Terminated())
	}
}

func (s *coffinTestSuite) TestMultipleWaitCalls() {
	s.cfn.Go("test", func() error {
		return nil
	})

	err := s.cfn.Wait()
	s.NoError(err)

	// calling Wait again should immediately return without errors
	err = s.cfn.Wait()
	s.NoError(err)
}

func (s *coffinTestSuite) TestCallWaitWithoutSpawningTask() {
	err := s.cfn.Wait()
	s.NoError(err)
	s.Equal(0, s.cfn.Running())
	s.Equal(0, s.cfn.Started())
	s.Equal(0, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestGoWithCanceledContext() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s.cfn.GoWithContext("test", func(ctx context.Context) error {
		<-ctx.Done()

		return ctx.Err()
	}, coffin.WithContext(ctx))

	err := s.cfn.Wait()
	s.EqualError(err, `failed to execute task "test" from package "coffin_test": context canceled`)
	s.Equal(0, s.cfn.Running())
	s.Equal(1, s.cfn.Started())
	s.Equal(1, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestWrappedError() {
	s.cfn.Go("test", func() error {
		return fmt.Errorf("some error")
	}, coffin.WithErrorWrapper("an error occurred = %v", true))

	err := s.cfn.Wait()
	s.EqualError(err, "an error occurred = true: some error")
	s.Equal(0, s.cfn.Running())
	s.Equal(1, s.cfn.Started())
	s.Equal(1, s.cfn.Terminated())
}

func (s *coffinTestSuite) TestCtxStaysSame() {
	ctx := s.cfn.Ctx()
	select {
	case <-ctx.Done():
		s.FailNow("should not yet be done with the context")
	default:
	}
	s.cfn.GoWithContext("exit immediately", func(context.Context) error {
		return nil
	})
	// check we can read from the context
	err := s.cfn.Wait()
	s.NoError(err)
	<-ctx.Done()
	<-s.cfn.Ctx().Done()
	s.Equal(ctx, s.cfn.Ctx())

	// run another go routine
	c := make(chan struct{})
	s.cfn.GoWithContext("exit once told", func(context.Context) error {
		<-c

		return nil
	})
	newCtx := s.cfn.Ctx()
	s.NotEqual(newCtx, ctx)
	select {
	case <-newCtx.Done():
		s.FailNow("should not yet be done with the new context")
	default:
	}
	close(c)
	err = s.cfn.Wait()
	s.NoError(err)
	<-newCtx.Done()
}

func (s *coffinTestSuite) TestTombStaysSame() {
	tmb := s.cfn.Entomb()
	select {
	case <-tmb.Dead():
		s.FailNow("tomb should not yet be dead")
	default:
	}
	s.cfn.GoWithContext("exit immediately", func(context.Context) error {
		return nil
	})
	// check we can read from the context
	err := s.cfn.Wait()
	s.NoError(err)
	<-tmb.Dead()
	<-s.cfn.Entomb().Dead()
	s.Equal(tmb, s.cfn.Entomb())

	// run another go routine
	c := make(chan struct{})
	s.cfn.GoWithContext("exit once told", func(context.Context) error {
		<-c

		return nil
	})
	newTmb := s.cfn.Entomb()
	s.NotEqual(newTmb, tmb)
	select {
	case <-newTmb.Dead():
		s.FailNow("new tomb should not yet be dead")
	default:
	}
	close(c)
	err = s.cfn.Wait()
	s.NoError(err)
	<-newTmb.Dead()
}
