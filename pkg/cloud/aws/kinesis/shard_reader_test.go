package kinesis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoKinesis "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type shardReaderTestSuite struct {
	suite.Suite

	ctx                context.Context
	stream             gosoKinesis.Stream
	shardId            gosoKinesis.ShardId
	logger             *logMocks.Logger
	metricWriter       *metricMocks.Writer
	metadataRepository *mocks.MetadataRepository
	kinesisClient      *mocks.Client
	settings           gosoKinesis.Settings
	clock              clock.FakeClock
	shardReader        gosoKinesis.ShardReader
	consumedRecords    [][]byte
	consumeRecordError error
}

func TestShardReader(t *testing.T) {
	suite.Run(t, new(shardReaderTestSuite))
}

func (s *shardReaderTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.stream = "testStream"
	s.shardId = "shard-007"
	s.metadataRepository = new(mocks.MetadataRepository)
	s.kinesisClient = new(mocks.Client)
	s.logger = new(logMocks.Logger)
	s.metricWriter = new(metricMocks.Writer)
	s.settings = gosoKinesis.Settings{
		WaitTime:         time.Second,
		PersistFrequency: time.Second * 10,
		ReleaseDelay:     time.Second * 30,
	}
	s.clock = clock.NewFakeClock()
	s.consumedRecords = nil
	s.consumeRecordError = nil

	s.shardReader = gosoKinesis.NewShardReaderWithInterfaces(s.stream, s.shardId, s.logger, s.metricWriter, s.metadataRepository, s.kinesisClient, s.settings, s.clock)
}

func (s *shardReaderTestSuite) TearDownTest() {
	s.logger.AssertExpectations(s.T())
	s.metricWriter.AssertExpectations(s.T())
	s.metadataRepository.AssertExpectations(s.T())
	s.kinesisClient.AssertExpectations(s.T())
}

func (s *shardReaderTestSuite) TestAcquireShardFails() {
	s.metadataRepository.On("AcquireShard", s.ctx, s.shardId).Return(nil, fmt.Errorf("fail")).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.EqualError(err, "failed to acquire shard: failed to acquire shard: fail")
}

func (s *shardReaderTestSuite) TestAcquireShardNotSuccessful() {
	// use a canceled context so we don't retry
	ctx, cancel := context.WithCancel(s.ctx)
	cancel()

	s.metadataRepository.On("AcquireShard", ctx, s.shardId).Return(nil, nil).Once()
	s.logger.On("Info", "could not acquire shard, leaving").Once()

	err := s.shardReader.Run(ctx, s.consumeRecord)
	s.NoError(err)
}

func (s *shardReaderTestSuite) TestGetShardIteratorFails() {
	checkpoint := new(mocks.Checkpoint)
	checkpoint.On("Persist", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(true, nil).Once()
	checkpoint.On("Release", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(nil).Once()
	checkpoint.On("GetSequenceNumber").Return(gosoKinesis.SequenceNumber("sequence number")).Once()
	defer checkpoint.AssertExpectations(s.T())

	s.metadataRepository.On("AcquireShard", s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.On("Info", "acquired shard").Once()
	s.logger.On("Info", "releasing shard").Once()
	s.kinesisClient.On("GetShardIterator", s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(nil, fmt.Errorf("fail")).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.EqualError(err, "failed to get shard iterator: failed to get shard iterator: fail")
}

func (s *shardReaderTestSuite) TestGetShardIteratorReturnsEmpty() {
	checkpoint := new(mocks.Checkpoint)
	checkpoint.On("Persist", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(true, nil).Once()
	checkpoint.On("Release", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(nil).Once()
	checkpoint.On("GetSequenceNumber").Return(gosoKinesis.SequenceNumber("")).Once()
	checkpoint.On("Done", gosoKinesis.SequenceNumber("")).Return(nil).Once()
	defer checkpoint.AssertExpectations(s.T())

	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Once()

	s.metadataRepository.On("AcquireShard", s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.On("Info", "acquired shard").Once()
	s.logger.On("Info", "releasing shard").Once()
	s.kinesisClient.On("GetShardIterator", s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:           aws.String(string(s.shardId)),
		ShardIteratorType: "TRIM_HORIZON",
		StreamName:        aws.String(string(s.stream)),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String(""),
	}, nil).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.NoError(err)
}

func (s *shardReaderTestSuite) TestGetRecordsAndReleaseFails() {
	checkpoint := new(mocks.Checkpoint)
	checkpoint.On("Persist", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(true, nil).Once()
	checkpoint.On("Release", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(fmt.Errorf("fail again")).Once()
	checkpoint.On("GetSequenceNumber").Return(gosoKinesis.SequenceNumber("LATEST")).Once()
	defer checkpoint.AssertExpectations(s.T())

	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Once()

	s.metadataRepository.On("AcquireShard", s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.On("Info", "acquired shard").Once()
	s.logger.On("Info", "releasing shard").Once()
	s.kinesisClient.On("GetShardIterator", s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:           aws.String(string(s.shardId)),
		ShardIteratorType: "LATEST",
		StreamName:        aws.String(string(s.stream)),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()
	s.kinesisClient.On("GetRecords", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Return(nil, fmt.Errorf("fail")).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.EqualError(err, multierror.Append(
		fmt.Errorf("failed reading records from shard: failed to get records from shard: fail"),
		fmt.Errorf("failed to release checkpoint for shard: fail again"),
	).Error())
}

func (s *shardReaderTestSuite) TestReleaseFailsAfterShardIteratorFailed() {
	checkpoint := new(mocks.Checkpoint)
	checkpoint.On("Persist", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(true, nil).Once()
	checkpoint.On("Release", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(fmt.Errorf("fail again")).Once()
	checkpoint.On("GetSequenceNumber").Return(gosoKinesis.SequenceNumber("sequence number")).Once()
	defer checkpoint.AssertExpectations(s.T())

	s.metadataRepository.On("AcquireShard", s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.On("Info", "acquired shard").Once()
	s.logger.On("Info", "releasing shard").Once()
	s.kinesisClient.On("GetShardIterator", s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(nil, fmt.Errorf("fail")).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.EqualError(err, multierror.Append(
		fmt.Errorf("failed to get shard iterator: failed to get shard iterator: fail"),
		fmt.Errorf("failed to release checkpoint for shard: fail again"),
	).Error())
}

func (s *shardReaderTestSuite) TestConsumeTwoBatches() {
	checkpoint := new(mocks.Checkpoint)
	checkpoint.On("Persist", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(true, nil).Once()
	checkpoint.On("Release", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(nil).Once()
	checkpoint.On("GetSequenceNumber").Return(gosoKinesis.SequenceNumber("sequence number")).Once()
	checkpoint.On("Advance", gosoKinesis.SequenceNumber("seq 1")).Return(nil).Once()
	checkpoint.On("Advance", gosoKinesis.SequenceNumber("seq 2")).Return(nil).Once()
	checkpoint.On("Done", gosoKinesis.SequenceNumber("seq 2")).Return(nil).Once()
	defer checkpoint.AssertExpectations(s.T())

	s.mockMetricCall("MillisecondsBehind", 1000, metric.UnitMillisecondsMaximum).Once()
	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Twice()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Twice()
	s.mockMetricCall("ReadRecords", 1, metric.UnitCount).Twice()

	s.metadataRepository.On("AcquireShard", s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.On("Info", "acquired shard").Once()
	s.logger.On("Info", "releasing shard").Once()
	s.logger.On("Info", "processed batch of %d records", 1).Twice()
	s.kinesisClient.On("GetShardIterator", s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()
	s.kinesisClient.On("GetRecords", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Return(&kinesis.GetRecordsOutput{
		Records: []types.Record{
			{
				Data:           []byte("data 1"),
				SequenceNumber: aws.String("seq 1"),
			},
		},
		MillisBehindLatest: aws.Int64(1000),
		NextShardIterator:  aws.String("next iterator"),
	}, nil).Once()
	s.kinesisClient.On("GetRecords", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("next iterator"),
		Limit:         aws.Int32(10000),
	}).Return(&kinesis.GetRecordsOutput{
		Records: []types.Record{
			{
				Data:           []byte("data 2"),
				SequenceNumber: aws.String("seq 2"),
			},
		},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String(""),
	}, nil).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.NoError(err)
	s.Equal([][]byte{
		[]byte("data 1"),
		[]byte("data 2"),
	}, s.consumedRecords)
}

func (s *shardReaderTestSuite) TestExpiredIteratorExceptionThenDelayedBadData() {
	checkpoint := new(mocks.Checkpoint)
	checkpoint.On("Persist", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(true, nil).Once()
	checkpoint.On("Release", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(nil).Once()
	checkpoint.On("GetSequenceNumber").Return(gosoKinesis.SequenceNumber("sequence number")).Twice()
	checkpoint.On("Advance", gosoKinesis.SequenceNumber("seq 1")).Return(nil).Once()
	defer checkpoint.AssertExpectations(s.T())

	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Times(3)
	s.mockMetricCall("FailedRecords", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Twice()
	s.mockMetricCall("ReadRecords", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadRecords", 0, metric.UnitCount).Once()

	s.metadataRepository.On("AcquireShard", s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.On("Info", "acquired shard").Once()
	s.logger.On("Info", "releasing shard").Once()
	s.logger.On("Info", "processed batch of %d records", 1).Once()
	s.logger.On("Info", "processed batch of %d records", 0).Once()
	s.logger.On("Error", "failed to handle record %s: %w", aws.String("seq 1"), fmt.Errorf("parse error"))
	s.kinesisClient.On("GetShardIterator", s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()
	s.kinesisClient.On("GetRecords", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Return(nil, &types.ExpiredIteratorException{}).Once()
	s.kinesisClient.On("GetShardIterator", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("new iterator"),
	}, nil).Once()
	s.kinesisClient.On("GetRecords", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("new iterator"),
		Limit:         aws.Int32(10000),
	}).Return(&kinesis.GetRecordsOutput{
		Records:            []types.Record{},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("next iterator"),
	}, nil).Run(func(args mock.Arguments) {
		go func() {
			s.clock.BlockUntilTimers(1)
			s.clock.Advance(time.Second)
		}()
	}).Once()
	s.kinesisClient.On("GetRecords", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("next iterator"),
		Limit:         aws.Int32(10000),
	}).Return(&kinesis.GetRecordsOutput{
		Records: []types.Record{
			{
				Data:           []byte("data 1"),
				SequenceNumber: aws.String("seq 1"),
			},
		},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("final iterator"),
	}, nil).Once()
	s.kinesisClient.On("GetRecords", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("final iterator"),
		Limit:         aws.Int32(10000),
	}).Return(nil, context.Canceled).Once()

	s.consumeRecordError = fmt.Errorf("parse error")

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.NoError(err)
	s.Equal([][]byte{
		[]byte("data 1"),
	}, s.consumedRecords)
}

func (s *shardReaderTestSuite) TestPersisterPersistCanceled() {
	checkpoint := new(mocks.Checkpoint)
	checkpoint.On("Persist", mock.AnythingOfType("*exec.manualCancelContext")).Run(func(args mock.Arguments) {
		go func() {
			s.clock.BlockUntilTimers(1)
			s.clock.Advance(time.Second)
		}()
	}).Return(false, context.Canceled).Once()
	checkpoint.On("Persist", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(true, nil).Once()
	checkpoint.On("Release", mock.AnythingOfType("*exec.DelayedCancelContext")).Return(nil).Once()
	checkpoint.On("GetSequenceNumber").Return(gosoKinesis.SequenceNumber("sequence number")).Once()
	defer checkpoint.AssertExpectations(s.T())

	go func() {
		s.clock.BlockUntilTickers(2)
		s.clock.Advance(time.Second * 10)
	}()

	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Twice()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadRecords", 0, metric.UnitCount).Once()

	s.metadataRepository.On("AcquireShard", s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.On("Info", "acquired shard").Once()
	s.logger.On("Info", "releasing shard").Once()
	s.logger.On("Info", "processed batch of %d records", 0).Once()
	s.kinesisClient.On("GetShardIterator", s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()
	s.kinesisClient.On("GetRecords", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Return(&kinesis.GetRecordsOutput{
		Records:            []types.Record{},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("next iterator"),
	}, nil).Once()
	s.kinesisClient.On("GetRecords", mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("next iterator"),
		Limit:         aws.Int32(10000),
	}).Return(nil, context.Canceled).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.NoError(err)
}

func (s *shardReaderTestSuite) consumeRecord(record []byte) error {
	s.consumedRecords = append(s.consumedRecords, record)

	return s.consumeRecordError
}

func (s *shardReaderTestSuite) mockMetricCall(metricName string, value float64, unit metric.StandardUnit) *mock.Call {
	return s.metricWriter.On("WriteOne", &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricName,
		Dimensions: metric.Dimensions{
			"StreamName": string(s.stream),
		},
		Value: value,
		Unit:  unit,
	})
}
