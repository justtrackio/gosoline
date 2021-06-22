package stream_test

import (
	"context"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	metricMocks "github.com/applike/gosoline/pkg/metric/mocks"
	"github.com/applike/gosoline/pkg/stream"
	streamMocks "github.com/applike/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type ProducerDaemonTestSuite struct {
	suite.Suite

	ctx        context.Context
	cancel     context.CancelFunc
	wait       chan error
	aggregator stream.ProducerDaemonAggregator
	output     *streamMocks.Output
	ticker     *clock.FakeTicker
	executor   exec.Executor
	daemon     producerDaemon
}

type producerDaemon interface {
	stream.Output
	kernel.Module
}

func (s *ProducerDaemonTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wait = make(chan error)
}

func (s *ProducerDaemonTestSuite) SetupDaemon(maxLogLevel int, batchSize int, aggregationSize int, interval time.Duration) {
	logger := logMocks.NewLoggerMockedUntilLevel(maxLogLevel)
	metric := metricMocks.NewWriterMockedAll()

	s.output = new(streamMocks.Output)
	s.ticker = clock.NewFakeTicker()
	s.executor = exec.NewBackoffExecutor(logger, &exec.ExecutableResource{
		Type: "test",
		Name: "test-output",
	}, &exec.BackoffSettings{
		Enabled:             true,
		Blocking:            true,
		CancelDelay:         time.Second,
		InitialInterval:     time.Millisecond * 50,
		RandomizationFactor: 0.5,
		Multiplier:          1.5,
		MaxInterval:         time.Second * 3,
		MaxElapsedTime:      time.Second * 15,
	}, exec.CheckRequestCanceled)

	tickerFactory := func(_ time.Duration) clock.Ticker {
		return s.ticker
	}

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

	s.daemon = stream.NewProducerDaemonWithInterfaces(logger, metric, s.aggregator, s.output, tickerFactory, "testDaemon", settings)

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

	call := s.output.On("Write", s.ctx, mock.Anything)
	call.Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		writtenMsg := args.Get(1)

		writtenJson, err := json.Marshal(writtenMsg)
		s.NoError(err)

		s.JSONEq(string(expectedJson), string(writtenJson))

		// simulate an executor like the real output would use
		_, err = s.executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
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
	s.output.AssertExpectations(s.T())
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

	err := s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteBatchOnClose() {
	s.SetupDaemon(log.PriorityInfo, 3, 1, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}

	err := s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	expected1 := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}
	s.expectMessage(expected1)

	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteBatchOnTick() {
	s.SetupDaemon(log.PriorityInfo, 3, 1, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}

	err := s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	expected1 := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}
	s.expectMessage(expected1)

	s.ticker.Trigger(time.Now())

	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
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

	err := s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	expected2 := []stream.WritableMessage{
		&stream.Message{Body: "3"},
	}
	s.expectMessage(expected2)

	s.ticker.Trigger(time.Now())
	time.Sleep(time.Millisecond)
	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteAggregate() {
	s.SetupDaemon(log.PriorityInfo, 2, 3, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
		&stream.Message{Body: "3"},
	}

	aggregateMessage, err := stream.MarshalJsonMessage(messages, map[string]interface{}{
		stream.AttributeAggregate: true,
	})
	s.NoError(err)

	expected := []stream.WritableMessage{aggregateMessage}
	s.expectMessage(expected)

	err = s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteWithCanceledError() {
	s.SetupDaemon(log.PriorityWarn, 5, 5, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}
	aggregateMessage, err := stream.MarshalJsonMessage(messages, map[string]interface{}{
		stream.AttributeAggregate: true,
	})
	s.NoError(err)

	expectedJson, err := json.Marshal([]stream.WritableMessage{aggregateMessage})
	s.NoError(err)

	s.output.On("Write", s.ctx, mock.Anything).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
		writtenMessages := args.Get(1)

		writtenJson, err := json.Marshal(writtenMessages)
		s.NoError(err)

		s.JSONEq(string(expectedJson), string(writtenJson))

		select {
		case _, ok := <-ctx.Done():
			s.False(ok, "expected the context to have been canceled")
		default:
			s.Fail("expected the context to have been canceled")
		}
	}).Return(context.Canceled).Once()

	_, err = stream.MarshalJsonMessage(messages, map[string]interface{}{
		stream.AttributeAggregate: true,
	})
	s.NoError(err)

	err = s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	err = s.stop()

	s.NoError(err)
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteAfterClose() {
	s.SetupDaemon(log.PriorityWarn, 1, 2, time.Hour)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}

	err := s.stop()

	s.NoError(err)

	err = s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	s.output.AssertExpectations(s.T())
}

func TestProducerDaemonTestSuite(t *testing.T) {
	suite.Run(t, new(ProducerDaemonTestSuite))
}
