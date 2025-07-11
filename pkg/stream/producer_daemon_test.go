package stream_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	streamMocks "github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ProducerDaemonTestSuite struct {
	suite.Suite

	ctx        context.Context
	cancel     context.CancelFunc
	wait       chan error
	aggregator stream.ProducerDaemonAggregator
	output     *streamMocks.Output
	clock      clock.FakeClock
	executor   exec.Executor
	daemon     producerDaemon
}

type producerDaemon interface {
	stream.Output
	kernel.Module
}

func (s *ProducerDaemonTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(s.T().Context())
	s.wait = make(chan error)
}

func (s *ProducerDaemonTestSuite) SetupDaemon(maxLogLevel int, batchSize int, aggregationSize int, interval time.Duration) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockUntilLevel(maxLogLevel))
	metric := metricMocks.NewWriterMockedAll()

	s.output = streamMocks.NewOutput(s.T())
	s.clock = clock.NewFakeClock()
	s.executor = exec.NewBackoffExecutor(logger, &exec.ExecutableResource{
		Type: "test",
		Name: "test-output",
	}, &exec.BackoffSettings{
		CancelDelay:     time.Millisecond * 100,
		InitialInterval: time.Millisecond * 50,
		MaxInterval:     time.Second * 3,
	}, []exec.ErrorChecker{exec.CheckRequestCanceled})

	settings := stream.ProducerDaemonSettings{
		Enabled:         true,
		Interval:        interval,
		BufferSize:      1,
		RunnerCount:     1,
		BatchSize:       batchSize,
		AggregationSize: aggregationSize,
	}

	var err error
	s.aggregator, err = stream.NewProducerDaemonAggregator(settings, stream.CompressionNone)
	s.NoError(err)

	s.daemon = stream.NewProducerDaemonWithInterfaces(logger, metric, s.aggregator, s.output, s.clock, "testDaemon", settings)

	running := make(chan struct{})

	go func() {
		close(running)
		err := s.daemon.Run(s.ctx)
		s.wait <- err
	}()

	<-running
}

func (s *ProducerDaemonTestSuite) stop() error {
	s.cancel()
	err := <-s.wait

	return err
}

func (s *ProducerDaemonTestSuite) expectMessage(messages []stream.WritableMessage) {
	expectedJson, err := json.Marshal(messages)
	s.NoError(err)

	call := s.output.EXPECT().Write(matcher.Context, mock.AnythingOfType("[]stream.WritableMessage"))
	call.Run(func(ctx context.Context, writtenMsg []stream.WritableMessage) {
		writtenJson, err := json.Marshal(writtenMsg)
		s.NoError(err)

		s.JSONEq(string(expectedJson), string(writtenJson))

		// simulate an executor like the real output would use
		_, err = s.executor.Execute(ctx, func(ctx context.Context) (any, error) {
			// return a context canceled error should the context already have been canceled (as the real output would)
			select {
			case <-ctx.Done():
				return nil, context.Canceled
			default:
				return nil, nil
			}
		})

		call.Return(err)
	}).Once()
}

func (s *ProducerDaemonTestSuite) TestRun() {
	s.SetupDaemon(log.PriorityInfo, 1, 1, time.Hour)
	err := s.stop()

	s.NoError(err, "there should be no error on run")
}

func (s *ProducerDaemonTestSuite) TestWriteBatch() {
	s.SetupDaemon(log.PriorityInfo, 2, 1, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
		&stream.Message{Body: "3"},
	}

	expected1 := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}
	expected2 := []stream.WritableMessage{
		&stream.Message{Body: "3"},
	}
	s.expectMessage(expected1)
	s.expectMessage(expected2)

	err := s.daemon.Write(s.T().Context(), messages)
	s.NoError(err, "there should be no error on write")

	err = s.stop()

	s.NoError(err, "there should be no error on run")
}

func (s *ProducerDaemonTestSuite) TestWriteBatchOnClose() {
	s.SetupDaemon(log.PriorityInfo, 3, 1, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}

	err := s.daemon.Write(s.T().Context(), messages)
	s.NoError(err, "there should be no error on write")

	expected1 := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}
	s.expectMessage(expected1)

	err = s.stop()

	s.NoError(err, "there should be no error on run")
}

func (s *ProducerDaemonTestSuite) TestWriteBatchOnTick() {
	s.SetupDaemon(log.PriorityInfo, 3, 1, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}

	err := s.daemon.Write(s.T().Context(), messages)
	s.NoError(err, "there should be no error on write")

	expected1 := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}
	s.expectMessage(expected1)

	s.clock.Advance(time.Hour)

	err = s.stop()

	s.NoError(err, "there should be no error on run")
}

func (s *ProducerDaemonTestSuite) TestWriteBatchOnTickAfterWrite() {
	s.SetupDaemon(log.PriorityInfo, 2, 1, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
		&stream.Message{Body: "3"},
	}

	expected1 := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}
	s.expectMessage(expected1)

	err := s.daemon.Write(s.T().Context(), messages)
	s.NoError(err, "there should be no error on write")

	expected2 := []stream.WritableMessage{
		&stream.Message{Body: "3"},
	}
	s.expectMessage(expected2)

	s.clock.Advance(time.Hour)
	time.Sleep(time.Millisecond)
	err = s.stop()

	s.NoError(err, "there should be no error on run")
}

func (s *ProducerDaemonTestSuite) TestWriteAggregate() {
	s.SetupDaemon(log.PriorityInfo, 2, 3, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
		&stream.Message{Body: "3"},
	}

	aggregateMessage, err := stream.MarshalJsonMessage(messages, map[string]string{
		stream.AttributeAggregate:      "true",
		stream.AttributeAggregateCount: "3",
	})
	s.NoError(err)

	expected := []stream.WritableMessage{aggregateMessage}
	s.expectMessage(expected)

	err = s.daemon.Write(s.T().Context(), messages)
	s.NoError(err, "there should be no error on write")

	err = s.stop()

	s.NoError(err, "there should be no error on run")
}

func (s *ProducerDaemonTestSuite) TestWriteAfterClose() {
	s.SetupDaemon(log.PriorityWarn, 1, 2, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}

	err := s.stop()
	s.NoError(err)

	err = s.daemon.Write(s.T().Context(), messages)
	s.EqualError(err, "can't write messages as the producer daemon testDaemon is not running")
}

func TestProducerDaemonTestSuite(t *testing.T) {
	suite.Run(t, new(ProducerDaemonTestSuite))
}
