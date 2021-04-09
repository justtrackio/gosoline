package stream_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mon"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	streamMocks "github.com/applike/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type ProducerDaemonTestSuite struct {
	suite.Suite

	ctx      context.Context
	cancel   context.CancelFunc
	wait     chan error
	output   *streamMocks.Output
	ticker   *clock.FakeTicker
	executor exec.Executor
	daemon   *stream.ProducerDaemon
}

func (s *ProducerDaemonTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wait = make(chan error)
}

func (s *ProducerDaemonTestSuite) SetupDaemon(maxLogLevel string, batchSize int, aggregationSize int, interval time.Duration, marshaller stream.AggregateMarshaller) {
	logger := monMocks.NewLoggerMockedUntilLevel(maxLogLevel)
	metric := monMocks.NewMetricWriterMockedAll()

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

	s.daemon = stream.NewProducerDaemonWithInterfaces(logger, metric, s.output, tickerFactory, marshaller, "testDaemon", stream.ProducerDaemonSettings{
		Enabled:         true,
		Interval:        interval,
		BufferSize:      1,
		RunnerCount:     1,
		BatchSize:       batchSize,
		AggregationSize: aggregationSize,
	})

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

func (s *ProducerDaemonTestSuite) expectMessage(msg []stream.WritableMessage) {
	call := s.output.On("Write", s.ctx, msg)
	call.Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)

		// simulate an executor like the real output would use
		_, err := s.executor.Execute(ctx, func(ctx context.Context) (interface{}, error) {
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
	s.SetupDaemon(mon.Info, 1, 1, time.Hour, stream.MarshalJsonMessage)
	err := s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteBatch() {
	s.SetupDaemon(mon.Info, 2, 1, time.Hour, stream.MarshalJsonMessage)

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
	s.SetupDaemon(mon.Info, 3, 1, time.Hour, stream.MarshalJsonMessage)

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
	s.SetupDaemon(mon.Info, 3, 1, time.Hour, stream.MarshalJsonMessage)

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
	s.SetupDaemon(mon.Info, 2, 1, time.Hour, stream.MarshalJsonMessage)

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
	s.SetupDaemon(mon.Info, 2, 3, time.Hour, stream.MarshalJsonMessage)

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

func (s *ProducerDaemonTestSuite) TestAggregateErrorOnWrite() {
	s.SetupDaemon(mon.Info, 2, 3, time.Hour, func(body interface{}, attributes ...map[string]interface{}) (*stream.Message, error) {
		return nil, fmt.Errorf("aggregate marshal error")
	})

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
		&stream.Message{Body: "3"},
	}

	_, err := stream.MarshalJsonMessage(messages, map[string]interface{}{
		stream.AttributeAggregate: true,
	})
	s.NoError(err)

	err = s.daemon.Write(context.Background(), messages)
	s.EqualError(err, "can not apply aggregation in producer testDaemon: can not marshal aggregate: aggregate marshal error")

	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestAggregateErrorOnClose() {
	s.SetupDaemon(mon.Info, 2, 3, time.Hour, func(body interface{}, attributes ...map[string]interface{}) (*stream.Message, error) {
		return nil, fmt.Errorf("aggregate marshal error")
	})

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
	}

	_, err := stream.MarshalJsonMessage(messages, map[string]interface{}{
		stream.AttributeAggregate: true,
	})
	s.NoError(err)

	err = s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	err = s.stop()

	s.EqualError(err, "error on close: can not flush all messages: can not flush aggregation: can not marshal aggregate: aggregate marshal error")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteWithCanceledError() {
	s.SetupDaemon(mon.Warn, 5, 5, time.Hour, stream.MarshalJsonMessage)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}
	aggregateMessage, err := stream.MarshalJsonMessage(messages, map[string]interface{}{
		stream.AttributeAggregate: true,
	})
	s.NoError(err)

	expected := []stream.WritableMessage{aggregateMessage}
	s.output.On("Write", s.ctx, expected).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)
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
	s.SetupDaemon(mon.Warn, 1, 2, time.Hour, stream.MarshalJsonMessage)

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
