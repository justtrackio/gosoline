package kinesis_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoKinesis "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis/mocks"
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockedMessage struct {
	data  []byte
	delay time.Duration
}

type mockedShardReader struct {
	messages      []mockedMessage
	waitForCancel bool
	err           error
}

type kinsumerTestSuite struct {
	suite.Suite

	ctx                context.Context
	logger             logMocks.LoggerMock
	stream             gosoKinesis.Stream
	kinesisClient      *mocks.Client
	metadataRepository *mocks.MetadataRepository
	metricWriter       *metricMocks.Writer
	clock              clock.FakeClock
	healthCheckTimer   clock.HealthCheckTimer
	handler            *mocks.MessageHandler
	kinsumer           gosoKinesis.Kinsumer

	shardReadersLck      *sync.Mutex
	shardReaders         map[gosoKinesis.ShardId][]*mocks.ShardReader
	expectedShardReaders map[gosoKinesis.ShardId][]mockedShardReader
	remainingForCancel   int
}

func TestKinsumer(t *testing.T) {
	suite.Run(t, new(kinsumerTestSuite))
}

func (s *kinsumerTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.logger = logMocks.NewLoggerMock(logMocks.WithTestingT(s.T()))
	s.stream = "gosoline-test-unitTest-kinesisTest-testData"
	s.kinesisClient = mocks.NewClient(s.T())
	s.metadataRepository = mocks.NewMetadataRepository(s.T())
	s.metricWriter = metricMocks.NewWriter(s.T())
	s.clock = clock.NewFakeClock()
	s.healthCheckTimer = clock.NewHealthCheckTimerWithInterfaces(s.clock, time.Minute)
	s.handler = mocks.NewMessageHandler(s.T())
	s.shardReadersLck = &sync.Mutex{}
	s.shardReaders = map[gosoKinesis.ShardId][]*mocks.ShardReader{}
	s.expectedShardReaders = map[gosoKinesis.ShardId][]mockedShardReader{}
	s.remainingForCancel = 0

	settings := gosoKinesis.Settings{
		AppId: cfg.AppId{
			Project:     "gosoline",
			Environment: "test",
			Family:      "unitTest",
			Application: "kinesisTest",
		},
		StreamName:        "testData",
		DiscoverFrequency: time.Second * 15,
		ReleaseDelay:      time.Second * 5,
		KeepShardOrder:    true,
	}

	s.kinsumer = gosoKinesis.NewKinsumerWithInterfaces(
		s.logger,
		settings,
		s.stream,
		s.kinesisClient,
		s.metadataRepository,
		s.metricWriter,
		s.clock,
		s.healthCheckTimer,
		func(logger log.Logger, shardId gosoKinesis.ShardId) gosoKinesis.ShardReader {
			s.shardReadersLck.Lock()
			defer s.shardReadersLck.Unlock()

			s.Contains(s.expectedShardReaders, shardId)

			shardReader := mocks.NewShardReader(s.T())
			s.shardReaders[shardId] = append(s.shardReaders[shardId], shardReader)
			mockedReader := s.expectedShardReaders[shardId][len(s.shardReaders[shardId])-1]
			shardReader.EXPECT().Run(matcher.Context, mock.MatchedBy(func(h any) bool {
				_, ok := h.(func([]byte) error)

				return ok
			})).Run(func(ctx context.Context, handler func([]byte) error) {
				for _, msg := range mockedReader.messages {
					<-s.clock.NewTimer(msg.delay).Chan()
					err := handler(msg.data)
					s.NoError(err)
				}

				if mockedReader.waitForCancel {
					<-ctx.Done()
				}

				s.shardReadersLck.Lock()
				defer s.shardReadersLck.Unlock()

				s.remainingForCancel--
				if s.remainingForCancel == 0 {
					s.kinsumer.Stop()
				}
			}).Return(mockedReader.err).Once()

			return shardReader
		},
	)

	s.logger.EXPECT().Info("removing client registration").Once()
}

func (s *kinsumerTestSuite) TearDownTest() {
	s.shardReadersLck.Lock()
	defer s.shardReadersLck.Unlock()

	for expectedShard, expectedReaders := range s.expectedShardReaders {
		s.Contains(s.shardReaders, expectedShard)
		s.Len(s.shardReaders[expectedShard], len(expectedReaders))
	}
}

func (s *kinsumerTestSuite) TestRegisterClientFail() {
	s.metadataRepository.EXPECT().RegisterClient(s.ctx).Return(0, 0, fmt.Errorf("fail")).Once()
	s.metadataRepository.EXPECT().DeregisterClient(matcher.Context).Return(nil).Once()

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.EqualError(err, "failed to load first list of shard ids and register as client: failed to register as client: fail")
}

func (s *kinsumerTestSuite) TestRegisterClientDeregisterFailToo() {
	s.metadataRepository.EXPECT().RegisterClient(s.ctx).Return(0, 0, fmt.Errorf("fail")).Once()
	s.metadataRepository.EXPECT().DeregisterClient(matcher.Context).Return(fmt.Errorf("also fail")).Once()

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.EqualError(err, multierror.Append(
		fmt.Errorf("failed to load first list of shard ids and register as client: failed to register as client: fail"),
		fmt.Errorf("failed to deregister client: also fail"),
	).Error())
}

func (s *kinsumerTestSuite) TestInitialListShardsFail() {
	s.metadataRepository.EXPECT().RegisterClient(s.ctx).Return(0, 1, nil).Once()
	s.metadataRepository.EXPECT().DeregisterClient(matcher.Context).Return(nil).Once()

	s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 0).Once()

	s.kinesisClient.EXPECT().ListShards(s.ctx, &kinesis.ListShardsInput{
		StreamName: aws.String(string(s.stream)),
	}).Return(nil, fmt.Errorf("fail")).Once()

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.EqualError(err, "failed to load first list of shard ids and register as client: failed to load shards from kinesis: failed to list shards of stream: fail")
}

func (s *kinsumerTestSuite) TestInitialListShardsNoSuchStream() {
	s.metadataRepository.EXPECT().RegisterClient(s.ctx).Return(0, 1, nil).Once()
	s.metadataRepository.EXPECT().DeregisterClient(matcher.Context).Return(nil).Once()

	s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 0).Once()

	s.kinesisClient.EXPECT().ListShards(s.ctx, &kinesis.ListShardsInput{
		StreamName: aws.String(string(s.stream)),
	}).Return(nil, &types.ResourceNotFoundException{
		Message: aws.String("no such stream"),
	}).Once()

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.EqualError(err, "failed to load first list of shard ids and register as client: failed to load shards from kinesis: No such stream: gosoline-test-unitTest-kinesisTest-testData")
	expectedErr := &gosoKinesis.NoSuchStreamError{}
	s.True(errors.As(err, &expectedErr))
}

func (s *kinsumerTestSuite) TestInitialListShardsResourceInUse() {
	s.metadataRepository.EXPECT().RegisterClient(s.ctx).Return(0, 1, nil).Once()
	s.metadataRepository.EXPECT().DeregisterClient(matcher.Context).Return(nil).Once()

	s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 0).Once()

	s.kinesisClient.EXPECT().ListShards(s.ctx, &kinesis.ListShardsInput{
		StreamName: aws.String(string(s.stream)),
	}).Return(nil, &types.ResourceInUseException{
		Message: aws.String("resource not ready"),
	}).Once()

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.EqualError(err, "failed to load first list of shard ids and register as client: failed to load shards from kinesis: Stream is busy: gosoline-test-unitTest-kinesisTest-testData")
	expectedErr := &gosoKinesis.StreamBusyError{}
	s.True(errors.As(err, &expectedErr))
}

func (s *kinsumerTestSuite) TestInitialListShardsIterate() {
	s.metadataRepository.EXPECT().RegisterClient(s.ctx).Return(0, 1, nil).Once()
	s.metadataRepository.EXPECT().DeregisterClient(matcher.Context).Return(nil).Once()

	s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 0).Once()

	s.kinesisClient.EXPECT().ListShards(s.ctx, &kinesis.ListShardsInput{
		StreamName: aws.String(string(s.stream)),
	}).Return(&kinesis.ListShardsOutput{
		NextToken: aws.String("next token"),
		Shards: []types.Shard{
			{
				ShardId: aws.String("shard1"),
			},
		},
	}, nil).Once()
	s.kinesisClient.EXPECT().ListShards(s.ctx, &kinesis.ListShardsInput{
		NextToken: aws.String("next token"),
	}).Return(&kinesis.ListShardsOutput{
		NextToken: nil,
		Shards: []types.Shard{
			{
				ShardId: aws.String("shard2"),
			},
		},
	}, nil).Once()

	s.metadataRepository.EXPECT().IsShardFinished(s.ctx, gosoKinesis.ShardId("shard1")).Return(false, nil).Once()
	s.metadataRepository.EXPECT().IsShardFinished(s.ctx, gosoKinesis.ShardId("shard2")).Return(false, nil).Once()

	s.logger.EXPECT().Info("kinsumer started %d consumers for %d shards", 2, 2).Once()
	s.logger.EXPECT().Info("started consuming shard").Twice()
	s.logger.EXPECT().Info("done consuming shard").Twice()
	s.mockShard("shard1", false, nil)
	s.mockShard("shard2", false, nil)

	s.logger.EXPECT().Info("leaving kinsumer").Once()
	s.logger.EXPECT().Info("stopping kinsumer").Once()
	s.handler.EXPECT().Done().Once()

	s.mockShardTaskRatio(200)

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.NoError(err)
}

func (s *kinsumerTestSuite) TestListShardsChangedShardIds() {
	s.mockBaseSuccess("shard1", "shard2")
	s.mockShardTaskRatio(200)
	s.mockShard("shard1", true, nil)
	s.mockShard("shard2", true, nil)

	go func() {
		s.clock.BlockUntilTickers(2)

		s.metadataRepository.EXPECT().RegisterClient(matcher.Context).Return(0, 1, nil).Once()
		s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 2).Once()
		s.logger.EXPECT().Info("discovered new shards or clients, restarting consumers for %d shards", 2).Once()
		s.logger.EXPECT().Info("started consuming shard").Twice()
		s.logger.EXPECT().Info("done consuming shard").Twice()
		s.logger.EXPECT().Info("kinsumer started %d consumers for %d shards", 2, 2).Once()
		s.mockShard("shard3", false, nil)
		s.mockShard("shard4", false, nil)

		s.mockShardTaskRatio(200)

		s.kinesisClient.EXPECT().ListShards(matcher.Context, &kinesis.ListShardsInput{
			StreamName: aws.String(string(s.stream)),
		}).Return(&kinesis.ListShardsOutput{
			NextToken: nil,
			Shards: []types.Shard{
				{
					ShardId: aws.String("shard4"),
				},
				{
					ShardId: aws.String("shard3"),
				},
			},
		}, nil).Once()
		s.metadataRepository.EXPECT().IsShardFinished(matcher.Context, gosoKinesis.ShardId("shard4")).Return(false, nil).Once()
		s.metadataRepository.EXPECT().IsShardFinished(matcher.Context, gosoKinesis.ShardId("shard3")).Return(false, nil).Once()

		s.clock.Advance(time.Second * 15)
	}()

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.NoError(err)
}

func (s *kinsumerTestSuite) TestShardListFinishedShardHandling() {
	s.metadataRepository.EXPECT().RegisterClient(s.ctx).Return(0, 1, nil).Once()
	s.metadataRepository.EXPECT().DeregisterClient(matcher.Context).Return(nil).Once()

	s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 0).Once()

	s.kinesisClient.EXPECT().ListShards(s.ctx, &kinesis.ListShardsInput{
		StreamName: aws.String(string(s.stream)),
	}).Return(&kinesis.ListShardsOutput{
		NextToken: nil,
		Shards: []types.Shard{
			{
				ShardId:       mdl.Box("finished shard with no parent"),
				ParentShardId: nil,
			},
			{
				ShardId:       mdl.Box("finished shard with parent"),
				ParentShardId: mdl.Box("finished shard with no parent"),
			},
			{
				ShardId:       mdl.Box("unfinished shard with no parent"),
				ParentShardId: nil,
			},
			{
				ShardId:       mdl.Box("unfinished shard with non-existing parent"),
				ParentShardId: mdl.Box("doesn't exist"),
			},
			{
				ShardId:       mdl.Box("unfinished shard with unfinished parent"),
				ParentShardId: mdl.Box("unfinished shard with no parent"),
			},
			{
				ShardId:       mdl.Box("unfinished shard with finished parent"),
				ParentShardId: mdl.Box("finished shard with no parent"),
			},
		},
	}, nil).Once()

	s.metadataRepository.EXPECT().IsShardFinished(s.ctx, gosoKinesis.ShardId("finished shard with no parent")).Return(true, nil).Once()
	s.metadataRepository.EXPECT().IsShardFinished(s.ctx, gosoKinesis.ShardId("finished shard with parent")).Return(true, nil).Once()
	s.metadataRepository.EXPECT().IsShardFinished(s.ctx, gosoKinesis.ShardId("unfinished shard with no parent")).Return(false, nil).Once()
	s.metadataRepository.EXPECT().IsShardFinished(s.ctx, gosoKinesis.ShardId("unfinished shard with non-existing parent")).Return(false, nil).Once()
	s.metadataRepository.EXPECT().IsShardFinished(s.ctx, gosoKinesis.ShardId("unfinished shard with unfinished parent")).Return(false, nil).Once()
	s.metadataRepository.EXPECT().IsShardFinished(s.ctx, gosoKinesis.ShardId("unfinished shard with finished parent")).Return(false, nil).Once()

	s.logger.EXPECT().Info("kinsumer started %d consumers for %d shards", 3, 3).Once()
	s.logger.EXPECT().Info("started consuming shard").Times(3)
	s.logger.EXPECT().Info("done consuming shard").Times(3)

	s.logger.EXPECT().Info("leaving kinsumer").Once()
	s.logger.EXPECT().Info("stopping kinsumer").Once()
	s.handler.EXPECT().Done().Once()

	s.mockShardTaskRatio(300)
	s.mockShard("unfinished shard with no parent", false, context.Canceled)
	s.mockShard("unfinished shard with non-existing parent", false, context.Canceled)
	s.mockShard("unfinished shard with finished parent", false, context.Canceled)

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.EqualError(err, "failed to consume from shard: context canceled")
}

func (s *kinsumerTestSuite) TestListShardsNoChangeThenCancel() {
	s.mockBaseSuccess("shard1")
	s.mockShardTaskRatio(100)
	s.mockShard("shard1", true, nil)

	go func() {
		s.clock.BlockUntilTickers(2)

		s.metadataRepository.EXPECT().RegisterClient(matcher.Context).Return(0, 1, nil).Once()
		s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 1).Once()

		s.kinesisClient.EXPECT().ListShards(matcher.Context, &kinesis.ListShardsInput{
			StreamName: aws.String(string(s.stream)),
		}).Return(&kinesis.ListShardsOutput{
			NextToken: nil,
			Shards: []types.Shard{
				{
					ShardId: aws.String("shard1"),
				},
			},
		}, nil).Run(func(ctx context.Context, params *kinesis.ListShardsInput, optFns ...func(*kinesis.Options)) {
			s.metadataRepository.EXPECT().RegisterClient(matcher.Context).Return(0, 1, nil).Once()
			s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 1).Once()

			s.kinesisClient.EXPECT().ListShards(matcher.Context, &kinesis.ListShardsInput{
				StreamName: aws.String(string(s.stream)),
			}).Return(nil, context.Canceled).Once()

			s.metadataRepository.EXPECT().IsShardFinished(matcher.Context, gosoKinesis.ShardId("shard1")).Return(false, nil).Once()

			s.clock.Advance(time.Second * 15)
		}).Once()

		s.clock.Advance(time.Second * 15)
	}()

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.NoError(err)
}

func (s *kinsumerTestSuite) TestListShardsFailOnRefresh() {
	s.mockBaseSuccess("shard1")
	s.mockShardTaskRatio(100)
	s.mockShard("shard1", true, nil)

	go func() {
		s.clock.BlockUntilTickers(2)

		s.metadataRepository.EXPECT().RegisterClient(matcher.Context).Return(0, 1, nil).Once()
		s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 1).Once()

		s.kinesisClient.EXPECT().ListShards(matcher.Context, &kinesis.ListShardsInput{
			StreamName: aws.String(string(s.stream)),
		}).Return(nil, fmt.Errorf("fail")).Once()

		s.clock.Advance(time.Second * 15)
	}()

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.EqualError(err, "failed to refresh shards: failed to load shards from kinesis: failed to list shards of stream: fail")
}

func (s *kinsumerTestSuite) TestConsumeMessagesThenCancel() {
	s.mockBaseSuccess("shard1")
	s.mockShardTaskRatio(100)
	s.mockShard("shard1", false, context.Canceled)
	s.mockShardMessage("shard1", []byte("message 1"), time.Millisecond)
	s.mockShardMessage("shard1", []byte("message 2"), time.Millisecond*5)
	s.mockShardMessage("shard1", []byte("message 3"), time.Millisecond*10)

	s.handler.EXPECT().Handle([]byte("message 1")).Return(nil).Once()
	s.handler.EXPECT().Handle([]byte("message 2")).Return(nil).Once()
	s.handler.EXPECT().Handle([]byte("message 3")).Return(nil).Once()

	go func() {
		s.clock.BlockUntilTimers(1)
		s.clock.Advance(time.Millisecond)

		s.clock.BlockUntilTimers(1)
		s.clock.Advance(time.Millisecond * 5)

		s.clock.BlockUntilTimers(1)
		s.clock.Advance(time.Millisecond * 10)
	}()

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.EqualError(err, "failed to consume from shard: context canceled")
}

func (s *kinsumerTestSuite) TestConsumeMessagesFails() {
	s.mockBaseSuccess("shard1")
	s.mockShardTaskRatio(100)
	s.mockShard("shard1", false, fmt.Errorf("fail"))

	err := s.kinsumer.Run(s.ctx, s.handler)
	s.EqualError(err, "failed to consume from shard: fail")
}

func (s *kinsumerTestSuite) mockBaseSuccess(shards ...string) {
	s.metadataRepository.EXPECT().RegisterClient(s.ctx).Return(0, 1, nil).Once()
	s.metadataRepository.EXPECT().DeregisterClient(matcher.Context).Return(nil).Once()

	s.logger.EXPECT().Info("we are client %d / %d, refreshing %d shards", 1, 1, 0).Once()

	shardsSlice := make([]types.Shard, len(shards))
	for i, shard := range shards {
		shardsSlice[i] = types.Shard{
			ShardId: aws.String(shard),
		}

		s.metadataRepository.EXPECT().IsShardFinished(s.ctx, gosoKinesis.ShardId(shard)).Return(false, nil).Once()
	}

	s.kinesisClient.EXPECT().ListShards(s.ctx, &kinesis.ListShardsInput{
		StreamName: aws.String(string(s.stream)),
	}).Return(&kinesis.ListShardsOutput{
		NextToken: nil,
		Shards:    shardsSlice,
	}, nil).Once()

	s.logger.EXPECT().Info("kinsumer started %d consumers for %d shards", len(shards), len(shards)).Once()
	s.logger.EXPECT().Info("started consuming shard").Times(len(shards))
	s.logger.EXPECT().Info("done consuming shard").Times(len(shards))

	s.logger.EXPECT().Info("leaving kinsumer").Once()
	s.logger.EXPECT().Info("stopping kinsumer").Once()
	s.handler.EXPECT().Done().Once()
}

func (s *kinsumerTestSuite) mockShardTaskRatio(taskShardRatio float64) {
	s.metricWriter.EXPECT().Write(metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: "ShardTaskRatioMax",
			Value:      taskShardRatio,
			Unit:       metric.UnitCountMaximum,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: "ShardTaskRatio",
			Dimensions: metric.Dimensions{
				"StreamName": string(s.stream),
			},
			Value: taskShardRatio,
			Unit:  metric.UnitCountAverage,
		},
	}).Once()
}

func (s *kinsumerTestSuite) mockShard(shardId gosoKinesis.ShardId, waitForCancel bool, err error) {
	s.shardReadersLck.Lock()
	defer s.shardReadersLck.Unlock()

	s.expectedShardReaders[shardId] = append(s.expectedShardReaders[shardId], mockedShardReader{
		messages:      nil,
		waitForCancel: waitForCancel,
		err:           err,
	})
	s.remainingForCancel++
}

func (s *kinsumerTestSuite) mockShardMessage(shardId gosoKinesis.ShardId, data []byte, delay time.Duration) {
	s.shardReadersLck.Lock()
	defer s.shardReadersLck.Unlock()

	readers := s.expectedShardReaders[shardId]
	readers[len(readers)-1].messages = append(readers[len(readers)-1].messages, mockedMessage{
		data:  data,
		delay: delay,
	})
}
