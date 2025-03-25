package coffin_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/stretchr/testify/suite"
)

type graveyardTestSuite struct {
	suite.Suite
	graveyard coffin.Graveyard
}

func TestGraveyard(t *testing.T) {
	suite.Run(t, new(graveyardTestSuite))
}

func (s *graveyardTestSuite) SetupTest() {
	s.graveyard = coffin.NewGraveyard(coffin.WithLabels(map[string]string{
		"test": s.T().Name(),
	}))
}

func (s *graveyardTestSuite) TestSpawn() {
	s.graveyard.Spawn("test", func() error {
		s.Equal(1, s.graveyard.Running())
		s.Equal(1, s.graveyard.Started())
		s.Equal(0, s.graveyard.Terminated())
		err := s.graveyard.Err()
		s.NoError(err)

		return nil
	})

	err := s.graveyard.Wait()
	s.NoError(err)
	s.Equal(0, s.graveyard.Running())
	s.Equal(1, s.graveyard.Started())
	s.Equal(1, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestSpawnWithError() {
	s.graveyard.Spawn("test", func() error {
		s.Equal(1, s.graveyard.Running())
		s.Equal(1, s.graveyard.Started())
		s.Equal(0, s.graveyard.Terminated())
		err := s.graveyard.Err()
		s.NoError(err)

		return fmt.Errorf("test error")
	})

	err := s.graveyard.Wait()
	s.EqualError(err, "test error")
	s.Equal(0, s.graveyard.Running())
	s.Equal(1, s.graveyard.Started())
	s.Equal(1, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestSpawnWithPanic() {
	s.graveyard.Spawn("test", func() error {
		s.Equal(1, s.graveyard.Running())
		s.Equal(1, s.graveyard.Started())
		s.Equal(0, s.graveyard.Terminated())
		err := s.graveyard.Err()
		s.NoError(err)

		panic("test panic")
	})

	err := s.graveyard.Wait()
	s.Error(err)
	s.Contains(err.Error(), "test panic")
	s.Equal(0, s.graveyard.Running())
	s.Equal(1, s.graveyard.Started())
	s.Equal(1, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestSpawnWithContext() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.graveyard.SpawnWithContext("test", ctx, func(ctxArg context.Context) error {
		s.Equal(ctx, ctxArg)

		s.Equal(1, s.graveyard.Running())
		s.Equal(1, s.graveyard.Started())
		s.Equal(0, s.graveyard.Terminated())
		err := s.graveyard.Err()
		s.NoError(err)

		return nil
	})

	err := s.graveyard.Wait()
	s.NoError(err)
	s.Equal(0, s.graveyard.Running())
	s.Equal(1, s.graveyard.Started())
	s.Equal(1, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestConcurrentSpawn() {
	for i := 0; i < 100; i++ {
		s.graveyard.Spawn(fmt.Sprintf("test-%02d", i), func() error {
			return nil
		}, coffin.WithLabels(map[string]string{
			"index": strconv.Itoa(i),
		}))
	}

	err := s.graveyard.Wait()
	s.NoError(err)
	s.Equal(0, s.graveyard.Running())
	s.Equal(100, s.graveyard.Started())
	s.Equal(100, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestMultipleErrors() {
	s.graveyard.Spawn("task1", func() error {
		return fmt.Errorf("error 1")
	})
	s.graveyard.Spawn("task2", func() error {
		return fmt.Errorf("error 2")
	})

	err := s.graveyard.Wait()
	s.Error(err)
	s.Contains(err.Error(), "error 1")
	s.Contains(err.Error(), "error 2")
	s.Equal(0, s.graveyard.Running())
	s.Equal(2, s.graveyard.Started())
	s.Equal(2, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestMixedSuccessFailure() {
	s.graveyard.Spawn("success-task", func() error {
		return nil
	})
	s.graveyard.Spawn("failure-task", func() error {
		return fmt.Errorf("task failed")
	})

	err := s.graveyard.Wait()
	s.EqualError(err, "task failed")
	s.Equal(0, s.graveyard.Running())
	s.Equal(2, s.graveyard.Started())
	s.Equal(2, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestNestedSpawn() {
	s.graveyard.Spawn("parent", func() error {
		s.graveyard.Spawn("child", func() error {
			return nil
		})

		return nil
	})

	err := s.graveyard.Wait()
	s.NoError(err)
	s.Equal(0, s.graveyard.Running())
	s.Equal(2, s.graveyard.Started())
	s.Equal(2, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestGraveyardReuse() {
	for i := 1; i <= 100; i++ {
		s.graveyard.Spawn("task", func() error {

			return nil
		})

		err := s.graveyard.Wait()
		s.NoError(err)
		s.Equal(0, s.graveyard.Running())
		s.Equal(i, s.graveyard.Started())
		s.Equal(i, s.graveyard.Terminated())
	}
}

func (s *graveyardTestSuite) TestMultipleWaitCalls() {
	s.graveyard.Spawn("test", func() error {
		return nil
	})

	err := s.graveyard.Wait()
	s.NoError(err)

	// calling Wait again should immediately return without errors
	err = s.graveyard.Wait()
	s.NoError(err)
}

func (s *graveyardTestSuite) TestCallWaitWithoutSpawningTask() {
	err := s.graveyard.Wait()
	s.NoError(err)
	s.Equal(0, s.graveyard.Running())
	s.Equal(0, s.graveyard.Started())
	s.Equal(0, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestSpawnWithCanceledContext() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	s.graveyard.SpawnWithContext("test", ctx, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	err := s.graveyard.Wait()
	s.EqualError(err, context.Canceled.Error())
	s.Equal(0, s.graveyard.Running())
	s.Equal(1, s.graveyard.Started())
	s.Equal(1, s.graveyard.Terminated())
}

func (s *graveyardTestSuite) TestWrappedError() {
	s.graveyard.Spawn("test", func() error {
		return fmt.Errorf("some error")
	}, coffin.WithErrorWrapper("an error occurred = %v", true))

	err := s.graveyard.Wait()
	s.EqualError(err, "an error occurred = true: some error")
	s.Equal(0, s.graveyard.Running())
	s.Equal(1, s.graveyard.Started())
	s.Equal(1, s.graveyard.Terminated())
}
