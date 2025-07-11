package exec_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sys/unix"
)

type ExecutorBackoffTestSuite struct {
	suite.Suite
	buildExecutor func(maxElapsedTime time.Duration, checkers ...exec.ErrorChecker) exec.Executor
}

func (s *ExecutorBackoffTestSuite) SetupTest() {
	s.buildExecutor = func(maxElapsedTime time.Duration, checkers ...exec.ErrorChecker) exec.Executor {
		resource := &exec.ExecutableResource{
			Type: "gosoline",
			Name: "test",
		}

		settings := &exec.BackoffSettings{
			CancelDelay:     0,
			InitialInterval: time.Millisecond,
			MaxInterval:     time.Millisecond * 2,
			MaxElapsedTime:  maxElapsedTime,
		}

		logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

		return exec.NewBackoffExecutor(logger, resource, settings, checkers)
	}
}

func (s *ExecutorBackoffTestSuite) TestPermanent() {
	tries := 0
	permanentError := fmt.Errorf("permanent error")

	checker := func(result any, err error) exec.ErrorType {
		return exec.ErrorTypePermanent
	}

	executor := s.buildExecutor(time.Millisecond*25, checker)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
		tries++

		return nil, permanentError
	})

	s.EqualError(err, permanentError.Error())
	s.Equal(1, tries)
}

func (s *ExecutorBackoffTestSuite) TestOk() {
	tries := 0
	okError := fmt.Errorf("ok error")

	checker := func(result any, err error) exec.ErrorType {
		return exec.ErrorTypeOk
	}

	executor := s.buildExecutor(time.Millisecond*25, checker)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
		tries++

		return nil, okError
	})

	s.EqualError(err, okError.Error())
	s.Equal(1, tries)
}

func (s *ExecutorBackoffTestSuite) TestRetryable() {
	tries := 0
	retryableError := fmt.Errorf("ok retryable")

	checker := func(result any, err error) exec.ErrorType {
		return exec.ErrorTypeRetryable
	}

	executor := s.buildExecutor(time.Millisecond*100, checker)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
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

	checker := func(result any, err error) exec.ErrorType {
		return exec.ErrorTypeUnknown
	}

	executor := s.buildExecutor(time.Millisecond*25, checker)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
		tries++

		return nil, unknownError
	})

	s.EqualError(err, unknownError.Error())
	s.Equal(1, tries)
}

func (s *ExecutorBackoffTestSuite) TestMaxElapsedTimeReached() {
	tries := 0
	longTakingErr := fmt.Errorf("this error occurred after reaching max elapsed time")

	checker := func(result any, err error) exec.ErrorType {
		return exec.ErrorTypeRetryable
	}

	executor := s.buildExecutor(time.Millisecond*25, checker)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
		time.Sleep(time.Millisecond * 30)
		tries++

		return nil, longTakingErr
	})

	s.True(exec.IsErrMaxElapsedTimeExceeded(err))
	s.EqualError(errors.Unwrap(err), longTakingErr.Error())
	s.Equal(1, tries)
}

func (s *ExecutorBackoffTestSuite) TestUsedClosedConnection() {
	tries := 0
	client := exec.NewTestHttpClient(time.Minute, exec.Trips{
		exec.DoTrip(time.Millisecond, errors.New("use of closed network connection")),
		exec.DoTrip(time.Millisecond, errors.New("use of closed network connection")),
		exec.DoTrip(time.Millisecond, nil),
	})

	executor := s.buildExecutor(time.Millisecond*100, exec.CheckUsedClosedConnectionError)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
		tries++

		return client.Get("http://test.url")
	})

	s.NoError(err)
	s.Equal(3, tries)
}

func (s *ExecutorBackoffTestSuite) TestConnectionError() {
	tries := 0
	client := exec.NewTestHttpClient(time.Minute, exec.Trips{
		exec.DoTrip(time.Millisecond, io.EOF),
		exec.DoTrip(time.Millisecond, unix.ECONNREFUSED),
		exec.DoTrip(time.Millisecond, unix.ECONNRESET),
		exec.DoTrip(time.Millisecond, unix.EPIPE),
		exec.DoTrip(time.Millisecond, nil),
	})

	executor := s.buildExecutor(time.Millisecond*100, exec.CheckConnectionError)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
		tries++

		return client.Get("http://test.url")
	})

	s.NoError(err)
	s.Equal(5, tries)
}

func (s *ExecutorBackoffTestSuite) TestTimeOutError() {
	tries := 0
	client := exec.NewTestHttpClient(time.Minute, exec.Trips{
		exec.DoTrip(time.Millisecond, unix.ETIMEDOUT),
		exec.DoTrip(time.Millisecond, unix.ETIMEDOUT),
		exec.DoTrip(time.Millisecond, nil),
	})

	executor := s.buildExecutor(time.Millisecond*100, exec.CheckTimeoutError)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
		tries++

		return client.Get("http://test.url")
	})

	s.NoError(err)
	s.Equal(3, tries)
}

func (s *ExecutorBackoffTestSuite) TestClientTimeoutError() {
	tries := 0
	client := exec.NewTestHttpClient(time.Minute, exec.Trips{
		exec.DoTrip(time.Millisecond, errors.New("(Client.Timeout exceeded while awaiting headers)")),
		exec.DoTrip(time.Millisecond, errors.New("(Client.Timeout exceeded while awaiting headers)")),
		exec.DoTrip(time.Millisecond, nil),
	})

	executor := s.buildExecutor(time.Millisecond*100, exec.CheckClientAwaitHeaderTimeoutError)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
		tries++

		return client.Get("http://test.url")
	})

	s.NoError(err)
	s.Equal(3, tries)
}

func (s *ExecutorBackoffTestSuite) TestTlsHandshakeTimeoutError() {
	tries := 0
	client := exec.NewTestHttpClient(time.Minute, exec.Trips{
		exec.DoTrip(time.Millisecond, errors.New("net/http: TLS handshake timeout")),
		exec.DoTrip(time.Millisecond, errors.New("net/http: TLS handshake timeout")),
		exec.DoTrip(time.Millisecond, nil),
	})

	executor := s.buildExecutor(time.Millisecond*100, exec.CheckTlsHandshakeTimeoutError)
	_, err := executor.Execute(s.T().Context(), func(ctx context.Context) (any, error) {
		tries++

		return client.Get("http://test.url")
	})

	s.NoError(err)
	s.Equal(3, tries)
}

func TestExecutorBackoffTestSuite(t *testing.T) {
	suite.Run(t, new(ExecutorBackoffTestSuite))
}
