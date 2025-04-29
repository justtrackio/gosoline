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
	"github.com/justtrackio/gosoline/pkg/mdl"
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
	healthCheckTimer   clock.HealthCheckTimer
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
	s.metadataRepository = mocks.NewMetadataRepository(s.T())
	s.kinesisClient = mocks.NewClient(s.T())
	s.logger = logMocks.NewLogger(s.T())
	s.metricWriter = metricMocks.NewWriter(s.T())
	s.settings = gosoKinesis.Settings{
		InitialPosition: gosoKinesis.SettingsInitialPosition{
			Type: types.ShardIteratorTypeLatest,
		},
		MaxBatchSize:     10_000,
		WaitTime:         time.Second,
		PersistFrequency: time.Second * 10,
		ReleaseDelay:     time.Second * 30,
	}
	s.clock = clock.NewFakeClock()
	s.healthCheckTimer = clock.NewHealthCheckTimerWithInterfaces(s.clock, time.Minute)
	s.consumedRecords = nil
	s.consumeRecordError = nil
}

func (s *shardReaderTestSuite) setupReader() {
	s.shardReader = gosoKinesis.NewShardReaderWithInterfaces(
		s.stream,
		s.shardId,
		s.logger,
		s.metricWriter,
		s.metadataRepository,
		s.kinesisClient,
		s.settings,
		s.clock,
		s.healthCheckTimer,
	)
}

func (s *shardReaderTestSuite) TestAcquireShardFails() {
	s.setupReader()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(nil, fmt.Errorf("fail")).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.EqualError(err, "failed to acquire shard: failed to acquire shard: fail")
}

func (s *shardReaderTestSuite) TestAcquireShardNotSuccessful() {
	s.setupReader()
	s.mockMetricCall("AcquireShardDelaySeconds", 0.0, metric.UnitSecondsMaximum)

	// use a canceled context so we don't retry
	ctx, cancel := context.WithCancel(s.ctx)
	cancel()

	s.metadataRepository.EXPECT().AcquireShard(ctx, s.shardId).Return(nil, nil).Once()
	s.logger.EXPECT().Info("could not acquire shard, leaving").Once()

	err := s.shardReader.Run(ctx, s.consumeRecord)
	s.NoError(err)
}

func (s *shardReaderTestSuite) TestGetShardIteratorFails() {
	s.setupReader()

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("sequence number").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.kinesisClient.EXPECT().GetShardIterator(s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(nil, fmt.Errorf("fail")).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.EqualError(err, "failed to get shard iterator: failed to get shard iterator: fail")
}

func (s *shardReaderTestSuite) TestGetShardIteratorReturnsEmpty() {
	s.setupReader()

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()
	checkpoint.EXPECT().Done(gosoKinesis.SequenceNumber("")).Return(nil).Once()

	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Once()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.kinesisClient.EXPECT().GetShardIterator(s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:           aws.String(string(s.shardId)),
		ShardIteratorType: "LATEST",
		StreamName:        aws.String(string(s.stream)),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String(""),
	}, nil).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.NoError(err)
}

func (s *shardReaderTestSuite) TestGetRecordsAndReleaseFails() {
	s.setupReader()

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(fmt.Errorf("fail again")).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()

	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Once()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.kinesisClient.EXPECT().GetShardIterator(s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:           aws.String(string(s.shardId)),
		ShardIteratorType: "LATEST",
		StreamName:        aws.String(string(s.stream)),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()
	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
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
	s.setupReader()

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(fmt.Errorf("fail again")).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("sequence number").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.kinesisClient.EXPECT().GetShardIterator(s.ctx, &kinesis.GetShardIteratorInput{
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
	s.setupReader()

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("sequence number").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber("seq 1"), gosoKinesis.ShardIterator("")).Return(nil).Once()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber("seq 2"), gosoKinesis.ShardIterator("")).Return(nil).Once()
	checkpoint.EXPECT().Done(gosoKinesis.SequenceNumber("seq 2")).Return(nil).Once()

	s.mockMetricCall("ProcessDuration", 0, metric.UnitMillisecondsAverage).Twice()
	s.mockMetricCall("MillisecondsBehind", 1000, metric.UnitMillisecondsMaximum).Once()
	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Twice()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Twice()
	s.mockMetricCall("ReadRecords", 1, metric.UnitCount).Twice()
	s.mockMetricCall("WaitDuration", 0, metric.UnitMillisecondsAverage).Once()
	s.mockMetricCall("WaitDuration", 1000, metric.UnitMillisecondsAverage).Once()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.logger.EXPECT().WithChannel("kinsumer-read").Return(s.logger)
	s.logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(s.logger)
	s.logger.EXPECT().Info("processed batch of %d records in %s", 1, mock.AnythingOfType("time.Duration")).Twice()

	s.kinesisClient.EXPECT().GetShardIterator(s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Run(func(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) {
		s.clock.Advance(s.settings.WaitTime)
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

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
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

func (s *shardReaderTestSuite) TestConsumeStartFromConsumeEmptyStream() {
	s.setupReader()

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("").Once()
	checkpoint.EXPECT().GetShardIterator().Return("initial iterator").Once()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber(""), gosoKinesis.ShardIterator("shard iterator")).Return(nil).Once()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber(""), gosoKinesis.ShardIterator("next iterator")).Return(nil).Once()
	checkpoint.EXPECT().Done(gosoKinesis.SequenceNumber("")).Return(nil).Once()

	s.mockMetricCall("ProcessDuration", 0, metric.UnitMillisecondsAverage).Twice()
	s.mockMetricCall("MillisecondsBehind", 1000, metric.UnitMillisecondsMaximum).Once()
	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Twice()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Twice()
	s.mockMetricCall("ReadRecords", 0, metric.UnitCount).Twice()
	s.mockMetricCall("WaitDuration", 0, metric.UnitMillisecondsAverage).Once()
	s.mockMetricCall("WaitDuration", 1000, metric.UnitMillisecondsAverage).Once()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.logger.EXPECT().WithChannel("kinsumer-read").Return(s.logger)
	s.logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(s.logger)
	s.logger.EXPECT().Info("processed batch of %d records in %s", 0, mock.AnythingOfType("time.Duration")).Twice()

	s.kinesisClient.EXPECT().GetRecords(s.ctx, &kinesis.GetRecordsInput{
		ShardIterator: aws.String("initial iterator"),
		Limit:         aws.Int32(1),
	}).Return(&kinesis.GetRecordsOutput{
		Records:           []types.Record{},
		NextShardIterator: aws.String("shard iterator"),
	}, nil).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Run(func(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) {
		s.clock.Advance(s.settings.WaitTime)
	}).Return(&kinesis.GetRecordsOutput{
		Records:            []types.Record{},
		MillisBehindLatest: aws.Int64(1000),
		NextShardIterator:  aws.String("next iterator"),
	}, nil).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("next iterator"),
		Limit:         aws.Int32(10000),
	}).Return(&kinesis.GetRecordsOutput{
		Records:            []types.Record{},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String(""),
	}, nil).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.NoError(err)
	s.Equal([][]byte(nil), s.consumedRecords)
}

func (s *shardReaderTestSuite) TestExpiredIteratorExceptionThenDelayedBadData() {
	s.setupReader()

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("sequence number").Twice()
	checkpoint.EXPECT().GetShardIterator().Return("").Twice()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber("sequence number"), gosoKinesis.ShardIterator("new iterator")).Return(nil).Once()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber("seq 1"), gosoKinesis.ShardIterator("")).Return(nil).Once()

	s.mockMetricCall("ProcessDuration", 0, metric.UnitMillisecondsAverage).Twice()
	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Times(3)
	s.mockMetricCall("FailedRecords", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Twice()
	s.mockMetricCall("ReadRecords", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadRecords", 0, metric.UnitCount).Once()
	s.mockMetricCall("WaitDuration", 1000, metric.UnitMillisecondsAverage).Twice()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.logger.EXPECT().WithChannel("kinsumer-read").Return(s.logger)
	s.logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(s.logger)
	s.logger.EXPECT().Info("processed batch of %d records in %s", 1, mock.AnythingOfType("time.Duration")).Once()
	s.logger.EXPECT().Info("processed batch of %d records in %s", 0, mock.AnythingOfType("time.Duration")).Once()
	s.logger.EXPECT().Error("failed to handle record %s: %w", aws.String("seq 1"), fmt.Errorf("parse error"))

	s.kinesisClient.EXPECT().GetShardIterator(s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Return(nil, &types.ExpiredIteratorException{}).Once()

	s.kinesisClient.EXPECT().GetShardIterator(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("new iterator"),
	}, nil).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("new iterator"),
		Limit:         aws.Int32(10000),
	}).Return(&kinesis.GetRecordsOutput{
		Records:            []types.Record{},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("next iterator"),
	}, nil).Run(func(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) {
		testClock := s.clock
		go func() {
			testClock.BlockUntilTimers(1)
			testClock.Advance(time.Second)
		}()
	}).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("next iterator"),
		Limit:         aws.Int32(10000),
	}).Run(func(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) {
		testClock := s.clock
		go func() {
			testClock.BlockUntilTimers(1)
			testClock.Advance(time.Second)
		}()
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

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
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
	s.setupReader()

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.manualCancelContext")).Return(false, context.Canceled).Maybe()
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("sequence number").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber("sequence number"), gosoKinesis.ShardIterator("shard iterator")).Return(nil).Once()

	s.mockMetricCall("ProcessDuration", 0, metric.UnitMillisecondsAverage).Once().Run(func(args mock.Arguments) {
		// need to wait with this call until we wrote the process duration - otherwise we could (in rare events) advance the
		// time while we measure the process duration, causing 10s instead of 0 to be said duration
		testClock := s.clock
		go func() {
			testClock.BlockUntilTickers(2)
			testClock.Advance(time.Second * 10)
		}()
	})
	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Twice()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadRecords", 0, metric.UnitCount).Once()
	s.mockMetricCall("WaitDuration", 0, metric.UnitMillisecondsAverage).Once()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.logger.EXPECT().WithChannel("kinsumer-read").Return(s.logger)
	s.logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(s.logger)
	s.logger.EXPECT().Info("processed batch of %d records in %s", 0, mock.AnythingOfType("time.Duration")).Once()
	s.kinesisClient.EXPECT().GetShardIterator(s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()
	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Run(func(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) {
		s.clock.Advance(time.Second)
	}).Return(&kinesis.GetRecordsOutput{
		Records:            []types.Record{},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("next iterator"),
	}, nil).Once()
	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("next iterator"),
		Limit:         aws.Int32(10000),
	}).Return(nil, context.Canceled).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.NoError(err)
}

func (s *shardReaderTestSuite) TestConsumeDelayWithWait() {
	s.settings.ConsumeDelay = time.Second
	s.setupReader()

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("sequence number").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber("seq 1"), gosoKinesis.ShardIterator("")).Return(nil).Once()
	checkpoint.EXPECT().Done(gosoKinesis.SequenceNumber("seq 1")).Return(nil).Once()

	s.mockMetricCall("SleepDuration", 1000, metric.UnitMillisecondsAverage).Once()
	s.mockMetricCall("ProcessDuration", 1000, metric.UnitMillisecondsAverage).Once()
	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Twice()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadRecords", 1, metric.UnitCount).Once()
	s.mockMetricCall("WaitDuration", 0, metric.UnitMillisecondsAverage).Once()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.logger.EXPECT().WithChannel("kinsumer-read").Return(s.logger)
	s.logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(s.logger)
	s.logger.EXPECT().Info("processed batch of %d records in %s", 1, mock.AnythingOfType("time.Duration")).Once()

	s.kinesisClient.EXPECT().GetShardIterator(s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Run(func(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) {
		testClock := s.clock
		go func() {
			testClock.BlockUntilTimers(1)
			testClock.Advance(time.Second)
		}()
	}).Return(&kinesis.GetRecordsOutput{
		Records: []types.Record{
			{
				Data:                        []byte("data 1"),
				SequenceNumber:              aws.String("seq 1"),
				ApproximateArrivalTimestamp: mdl.Box(s.clock.Now()),
			},
		},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String(""),
	}, nil).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.NoError(err)
	s.Equal([][]byte{
		[]byte("data 1"),
	}, s.consumedRecords)
}

func (s *shardReaderTestSuite) TestConsumeDelayWithOldRecord() {
	s.settings.ConsumeDelay = time.Second
	s.setupReader()

	recordArrivalTime := s.clock.Now()
	s.clock.Advance(time.Minute)

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("sequence number").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber("seq 1"), gosoKinesis.ShardIterator("")).Return(nil).Once()
	checkpoint.EXPECT().Done(gosoKinesis.SequenceNumber("seq 1")).Return(nil).Once()

	s.mockMetricCall("ProcessDuration", 0, metric.UnitMillisecondsAverage).Once()
	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Twice()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadRecords", 1, metric.UnitCount).Once()
	s.mockMetricCall("WaitDuration", 1000, metric.UnitMillisecondsAverage).Once()

	s.metadataRepository.EXPECT().AcquireShard(s.ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.logger.EXPECT().WithChannel("kinsumer-read").Return(s.logger)
	s.logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(s.logger)
	s.logger.EXPECT().Info("processed batch of %d records in %s", 1, mock.AnythingOfType("time.Duration")).Once()

	s.kinesisClient.EXPECT().GetShardIterator(s.ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Return(&kinesis.GetRecordsOutput{
		Records: []types.Record{
			{
				Data:                        []byte("data 1"),
				SequenceNumber:              aws.String("seq 1"),
				ApproximateArrivalTimestamp: mdl.Box(recordArrivalTime),
			},
		},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String(""),
	}, nil).Once()

	err := s.shardReader.Run(s.ctx, s.consumeRecord)
	s.NoError(err)
	s.Equal([][]byte{
		[]byte("data 1"),
	}, s.consumedRecords)
}

func (s *shardReaderTestSuite) TestConsumeDelayWithCancelDuringWait() {
	s.settings.ConsumeDelay = time.Minute
	s.settings.ReleaseDelay = time.Millisecond
	s.setupReader()

	ctx, cancel := context.WithCancel(s.ctx)

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("sequence number").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()

	s.mockMetricCall("ProcessDuration", 0, metric.UnitMillisecondsAverage).Once()
	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Twice()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadRecords", 0, metric.UnitCount).Once()
	s.mockMetricCall("WaitDuration", 1000, metric.UnitMillisecondsAverage).Once()

	s.metadataRepository.EXPECT().AcquireShard(ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.logger.EXPECT().WithChannel("kinsumer-read").Return(s.logger)
	s.logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(s.logger)
	s.logger.EXPECT().Info("processed batch of %d records in %s", 0, mock.AnythingOfType("time.Duration")).Once()

	s.kinesisClient.EXPECT().GetShardIterator(ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Run(func(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) {
		go func() {
			time.Sleep(time.Millisecond)
			cancel()
		}()
	}).Return(&kinesis.GetRecordsOutput{
		Records: []types.Record{
			{
				Data:                        []byte("data 1"),
				SequenceNumber:              aws.String("seq 1"),
				ApproximateArrivalTimestamp: mdl.Box(s.clock.Now()),
			},
		},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String(""),
	}, nil).Once()

	err := s.shardReader.Run(ctx, s.consumeRecord)
	if err != nil {
		// we have a race condition in the test - either we see the context get canceled before the coffin in the shard
		// reader finishes, and thus we get the canceled error, or the coffin finishes first, and we get no error at all...
		s.EqualError(err, "context canceled")
	}
	s.Nil(s.consumedRecords)
}

func (s *shardReaderTestSuite) TestConsumeDelayWithCancelDuringWaitNoRecords() {
	s.settings.ConsumeDelay = time.Minute
	s.settings.ReleaseDelay = time.Millisecond
	s.setupReader()

	ctx, cancel := context.WithCancel(s.ctx)

	checkpoint := mocks.NewCheckpoint(s.T())
	checkpoint.EXPECT().Persist(mock.AnythingOfType("*exec.stoppableContext")).Return(true, nil).Once()
	checkpoint.EXPECT().Release(mock.AnythingOfType("*exec.stoppableContext")).Return(nil).Once()
	checkpoint.EXPECT().GetSequenceNumber().Return("sequence number").Once()
	checkpoint.EXPECT().GetShardIterator().Return("").Once()
	checkpoint.EXPECT().Advance(gosoKinesis.SequenceNumber("sequence number"), gosoKinesis.ShardIterator("shard iterator")).Return(nil).Once()
	checkpoint.EXPECT().Done(gosoKinesis.SequenceNumber("sequence number")).Return(nil).Once()

	s.mockMetricCall("ProcessDuration", 0, metric.UnitMillisecondsAverage).Once()
	s.mockMetricCall("MillisecondsBehind", 0, metric.UnitMillisecondsMaximum).Twice()
	s.mockMetricCall("ReadCount", 1, metric.UnitCount).Once()
	s.mockMetricCall("ReadRecords", 0, metric.UnitCount).Once()
	s.mockMetricCall("WaitDuration", 1000, metric.UnitMillisecondsAverage).Once()

	s.metadataRepository.EXPECT().AcquireShard(ctx, s.shardId).Return(checkpoint, nil).Once()
	s.logger.EXPECT().Info("acquired shard").Once()
	s.logger.EXPECT().Info("releasing shard").Once()
	s.logger.EXPECT().WithChannel("kinsumer-read").Return(s.logger)
	s.logger.EXPECT().WithFields(mock.AnythingOfType("log.Fields")).Return(s.logger)
	s.logger.EXPECT().Info("processed batch of %d records in %s", 0, mock.AnythingOfType("time.Duration")).Once()

	s.kinesisClient.EXPECT().GetShardIterator(ctx, &kinesis.GetShardIteratorInput{
		ShardId:                aws.String(string(s.shardId)),
		ShardIteratorType:      "AFTER_SEQUENCE_NUMBER",
		StreamName:             aws.String(string(s.stream)),
		StartingSequenceNumber: aws.String("sequence number"),
	}).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("shard iterator"),
	}, nil).Once()

	s.kinesisClient.EXPECT().GetRecords(mock.AnythingOfType("*context.cancelCtx"), &kinesis.GetRecordsInput{
		ShardIterator: aws.String("shard iterator"),
		Limit:         aws.Int32(10000),
	}).Run(func(ctx context.Context, params *kinesis.GetRecordsInput, optFns ...func(*kinesis.Options)) {
		cancel()
	}).Return(&kinesis.GetRecordsOutput{
		Records:            []types.Record{},
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String(""),
	}, nil).Once()

	err := s.shardReader.Run(ctx, s.consumeRecord)
	s.NoError(err)
	s.Nil(s.consumedRecords)
}

func (s *shardReaderTestSuite) consumeRecord(record []byte) error {
	s.consumedRecords = append(s.consumedRecords, record)

	return s.consumeRecordError
}

func (s *shardReaderTestSuite) mockMetricCall(metricName string, value float64, unit metric.StandardUnit) *metricMocks.Writer_Write_Call {
	return s.metricWriter.EXPECT().Write(metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricName,
			Dimensions: metric.Dimensions{
				"StreamName": string(s.stream),
			},
			Value: value,
			Unit:  unit,
			Kind:  metric.KindTotal,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricName,
			Dimensions: metric.Dimensions{
				"StreamName": string(s.stream),
				"ShardId":    string(s.shardId),
			},
			Value: value,
			Unit:  unit,
		},
	})
}
