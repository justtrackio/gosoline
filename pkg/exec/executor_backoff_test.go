package exec_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type ExecutorBackoffTestSuite struct {
	suite.Suite
	buildExecutor func(...exec.ErrorChecker) exec.Executor
}

func (s *ExecutorBackoffTestSuite) SetupTest() {
	s.buildExecutor = func(checkers ...exec.ErrorChecker) exec.Executor {
		resource := &exec.ExecutableResource{
			Type: "gosoline",
			Name: "test",
		}

		settings := &exec.BackoffSettings{
			Enabled:         true,
			Blocking:        false,
			CancelDelay:     0,
			InitialInterval: time.Millisecond,
			MaxInterval:     time.Second,
			MaxElapsedTime:  time.Second,
		}

		logger := mocks.NewLoggerMockedAll()

		return exec.NewBackoffExecutor(logger, resource, settings, checkers...)
	}
}

func (s *ExecutorBackoffTestSuite) TestPermanent() {
	tries := 0
	permanentError := fmt.Errorf("permanent error")

	checker := func(result interface{}, err error) exec.ErrorType {
		return exec.ErrorPermanent
	}

	executor := s.buildExecutor(checker)
	_, err := executor.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		tries++
		return nil, permanentError
	})

	s.EqualError(err, permanentError.Error())
	s.Equal(1, tries)
}

func (s *ExecutorBackoffTestSuite) TestOk() {
	tries := 0
	okError := fmt.Errorf("ok error")

	checker := func(result interface{}, err error) exec.ErrorType {
		return exec.ErrorOk
	}

	executor := s.buildExecutor(checker)
	_, err := executor.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		tries++
		return nil, okError
	})

	s.EqualError(err, okError.Error())
	s.Equal(1, tries)
}

func (s *ExecutorBackoffTestSuite) TestRetryable() {
	tries := 0
	retryableError := fmt.Errorf("ok retryable")

	checker := func(result interface{}, err error) exec.ErrorType {
		return exec.ErrorRetryable
	}

	executor := s.buildExecutor(checker)
	_, err := executor.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		tries++

		if tries < 3 {
			return nil, retryableError
		}

		return nil, nil
	})

	s.NoError(err)
	s.Equal(3, tries)
}

func (s *ExecutorBackoffTestSuite) TestUnknown() {
	tries := 0
	unknownError := fmt.Errorf("unknown error")

	checker := func(result interface{}, err error) exec.ErrorType {
		return exec.ErrorUnknown
	}

	executor := s.buildExecutor(checker)
	_, err := executor.Execute(context.Background(), func(ctx context.Context) (interface{}, error) {
		tries++
		return nil, unknownError
	})

	s.EqualError(err, unknownError.Error())
	s.Equal(1, tries)
}

func TestExecutorBackoffTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorBackoffTestSuite))
}
