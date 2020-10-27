package stream_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	monMocks "github.com/applike/gosoline/pkg/mon/mocks"
	"github.com/applike/gosoline/pkg/stream"
	streamMocks "github.com/applike/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type ProducerDaemonTestSuite struct {
	suite.Suite

	ctx    context.Context
	cancel context.CancelFunc
	wait   chan error
	output *streamMocks.Output
	ticker *clock.FakeTicker
	daemon *stream.ProducerDaemon
}

func (s *ProducerDaemonTestSuite) SetupTest() {
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.wait = make(chan error)
}

func (s *ProducerDaemonTestSuite) SetupDaemon(batchSize int, aggregationSize int, interval time.Duration, marshaller stream.AggregateMarshaller) {
	logger := monMocks.NewLoggerMockedAll()
	metric := monMocks.NewMetricWriterMockedAll()

	s.output = new(streamMocks.Output)
	s.ticker = clock.NewFakeTicker()

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

func (s *ProducerDaemonTestSuite) TestRun() {
	s.SetupDaemon(1, 1, time.Hour, stream.MarshalJsonMessage)
	err := s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteBatch() {
	s.SetupDaemon(2, 1, time.Hour, stream.MarshalJsonMessage)

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
	s.output.On("Write", s.ctx, expected1).Return(nil)
	s.output.On("Write", s.ctx, expected2).Return(nil)

	err := s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteBatchOnClose() {
	s.SetupDaemon(3, 1, time.Hour, stream.MarshalJsonMessage)

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
	s.output.On("Write", s.ctx, expected1).Return(nil)

	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteBatchOnTick() {
	s.SetupDaemon(3, 1, time.Hour, stream.MarshalJsonMessage)

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
	s.output.On("Write", s.ctx, expected1).Return(nil)

	s.ticker.Trigger(time.Now())

	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteBatchOnTickAfterWrite() {
	s.SetupDaemon(2, 1, time.Hour, stream.MarshalJsonMessage)

	messages := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
		&stream.Message{Body: "3"},
	}

	expected1 := []stream.WritableMessage{
		&stream.Message{Body: "1"},
		&stream.Message{Body: "2"},
	}
	s.output.On("Write", s.ctx, expected1).Return(nil)

	err := s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	expected2 := []stream.WritableMessage{
		&stream.Message{Body: "3"},
	}
	s.output.On("Write", s.ctx, expected2).Return(nil)

	s.ticker.Trigger(time.Now())
	time.Sleep(time.Millisecond)
	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestWriteAggregate() {
	s.SetupDaemon(2, 3, time.Hour, stream.MarshalJsonMessage)

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
	s.output.On("Write", s.ctx, expected).Return(nil)

	err = s.daemon.Write(context.Background(), messages)
	s.NoError(err, "there should be no error on write")

	err = s.stop()

	s.NoError(err, "there should be no error on run")
	s.output.AssertExpectations(s.T())
}

func (s *ProducerDaemonTestSuite) TestAggregateErrorOnWrite() {
	s.SetupDaemon(2, 3, time.Hour, func(body interface{}, attributes ...map[string]interface{}) (*stream.Message, error) {
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
	s.SetupDaemon(2, 3, time.Hour, func(body interface{}, attributes ...map[string]interface{}) (*stream.Message, error) {
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

func TestProducerDaemonTestSuite(t *testing.T) {
	suite.Run(t, new(ProducerDaemonTestSuite))
}
