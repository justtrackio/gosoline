package stream_test

import (
	"context"
	"fmt"
	"testing"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/suite"
)

type producerDaemonPartitionedAggregatorTestSuite struct {
	suite.Suite
	ctx         context.Context
	logger      *logMocks.Logger
	rand        *mocks.PartitionerRand
	aggregators []*mocks.ProducerDaemonAggregator
	aggregator  stream.ProducerDaemonAggregator
}

func (s *producerDaemonPartitionedAggregatorTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.logger = logMocks.NewLoggerMock()
	s.rand = new(mocks.PartitionerRand)
	s.aggregators = []*mocks.ProducerDaemonAggregator{
		new(mocks.ProducerDaemonAggregator),
		new(mocks.ProducerDaemonAggregator),
		new(mocks.ProducerDaemonAggregator),
		new(mocks.ProducerDaemonAggregator),
	}
	var err error
	next := 0
	expectedExplicitHashKeys := []string{
		"42535295865117307932921825928971026431",  // floor((2 ^ 128 - 1) / 4 / 2)
		"127605887595351923798765477786913079294", // floor((2 ^ 128 - 1) / 4) * 1 + floor((2 ^ 128 - 1) / 4 / 2)
		"212676479325586539664609129644855132157", // floor((2 ^ 128 - 1) / 4) * 2 + floor((2 ^ 128 - 1) / 4 / 2)
		"297747071055821155530452781502797185020", // floor((2 ^ 128 - 1) / 4) * 3 + floor((2 ^ 128 - 1) / 4 / 2)
	}
	createAggregator := func(attributes map[string]interface{}) (stream.ProducerDaemonAggregator, error) {
		defer func() {
			next++
		}()

		s.Equal(map[string]interface{}{
			stream.AttributeKinesisExplicitHashKey: expectedExplicitHashKeys[next],
		}, attributes)

		return s.aggregators[next], nil
	}
	s.aggregator, err = stream.NewProducerDaemonPartitionedAggregatorWithInterfaces(s.logger, s.rand, 4, createAggregator)
	s.Equal(4, next)
	s.NoError(err)
}

func (s *producerDaemonPartitionedAggregatorTestSuite) TearDownTest() {
	s.logger.AssertExpectations(s.T())
	s.rand.AssertExpectations(s.T())
	for _, aggregator := range s.aggregators {
		aggregator.AssertExpectations(s.T())
	}
}

func (s *producerDaemonPartitionedAggregatorTestSuite) TestAggregatePlainMessage() {
	s.rand.On("Intn", 4).Return(0).Once()
	s.aggregators[0].On("Write", s.ctx, &stream.Message{}).Return(nil, nil).Once()

	flush, err := s.aggregator.Write(s.ctx, &stream.Message{})
	s.NoError(err)
	s.Nil(flush)
}

func (s *producerDaemonPartitionedAggregatorTestSuite) TestAggregatePartitionedMessage() {
	s.aggregators[2].On("Write", s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisPartitionKey: "my partition key",
		},
	}).Return(nil, nil).Once()

	flush, err := s.aggregator.Write(s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisPartitionKey: "my partition key", // md5sum ae926123d38f2789b5c350160d6e6c56 -> 6 % 4 = 2
		},
	})
	s.NoError(err)
	s.Nil(flush)
}

func (s *producerDaemonPartitionedAggregatorTestSuite) TestAggregateExplicitHashedMessage() {
	s.aggregators[2].On("Write", s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisExplicitHashKey: "232045716840113089107413691294511164502",
		},
	}).Return(nil, nil).Once()

	flush, err := s.aggregator.Write(s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisExplicitHashKey: "232045716840113089107413691294511164502", // "my partition key" md5 hashed and converted to base 10
		},
	})
	s.NoError(err)
	s.Nil(flush)
}

func (s *producerDaemonPartitionedAggregatorTestSuite) TestAggregateMixedMessages() {
	s.aggregators[2].On("Write", s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisPartitionKey:    "not my partition key",
			stream.AttributeKinesisExplicitHashKey: "232045716840113089107413691294511164502",
		},
	}).Return(nil, nil).Once()

	flush, err := s.aggregator.Write(s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisPartitionKey:    "not my partition key",                    // ignored
			stream.AttributeKinesisExplicitHashKey: "232045716840113089107413691294511164502", // "my partition key" md5 hashed and converted to base 10
		},
	})
	s.NoError(err)
	s.Nil(flush)
}

func (s *producerDaemonPartitionedAggregatorTestSuite) TestGettingExplicitHashKeyFails() {
	s.logger.On("Error", "failed to determine partition or explicit hash key, will choose one at random: %w", fmt.Errorf("invalid explicit hash key: not a number")).Once()
	s.rand.On("Intn", 4).Return(3).Once()
	s.aggregators[3].On("Write", s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisExplicitHashKey: "not a number",
		},
	}).Return(nil, nil).Once()

	flush, err := s.aggregator.Write(s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisExplicitHashKey: "not a number",
		},
	})
	s.NoError(err)
	s.Nil(flush)

}

func (s *producerDaemonPartitionedAggregatorTestSuite) TestGettingPartitionKeyFails() {
	s.logger.On("Error", "failed to determine partition or explicit hash key, will choose one at random: %w", fmt.Errorf("the type of the gosoline.kinesis.partitionKey attribute with value {} should be castable to string: %w", fmt.Errorf("unable to cast struct {}{} of type struct {} to string"))).Once()
	s.rand.On("Intn", 4).Return(1).Once()
	s.aggregators[1].On("Write", s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisPartitionKey: struct{}{},
		},
	}).Return(nil, nil).Once()

	flush, err := s.aggregator.Write(s.ctx, &stream.Message{
		Attributes: map[string]interface{}{
			stream.AttributeKinesisPartitionKey: struct{}{},
		},
	})
	s.NoError(err)
	s.Nil(flush)
}

func (s *producerDaemonPartitionedAggregatorTestSuite) TestFlushSuccess() {
	for i, aggregator := range s.aggregators {
		aggregator.On("Flush").Return([]stream.AggregateFlush{
			{
				MessageCount: i + 1,
			},
		}, nil).Once()
	}

	flushed, err := s.aggregator.Flush()
	s.NoError(err)
	s.Equal([]stream.AggregateFlush{
		{
			MessageCount: 1,
		},
		{
			MessageCount: 2,
		},
		{
			MessageCount: 3,
		},
		{
			MessageCount: 4,
		},
	}, flushed)
}

func (s *producerDaemonPartitionedAggregatorTestSuite) TestFlushFailure() {
	s.aggregators[0].On("Flush").Return([]stream.AggregateFlush{
		{
			MessageCount: 1,
		},
	}, nil).Once()
	s.aggregators[1].On("Flush").Return(nil, fmt.Errorf("fail")).Once()

	flushed, err := s.aggregator.Flush()
	s.Nil(flushed)
	s.EqualError(err, "failed to flush bucket: fail")
}

func TestProducerDaemonPartitionedAggregator(t *testing.T) {
	suite.Run(t, new(producerDaemonPartitionedAggregatorTestSuite))
}
