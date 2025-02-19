package stream_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	kinesisMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis/mocks"
	"github.com/justtrackio/gosoline/pkg/conc"
	concDdbMocks "github.com/justtrackio/gosoline/pkg/conc/ddb/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/stream/mocks"
	"github.com/stretchr/testify/suite"
)

func TestKinsumerAutoscaleModuleTestSuite(t *testing.T) {
	suite.Run(t, new(KinsumerAutoscaleModuleTestSuite))
}

type kinsumerAutoscaleModuleTestCase struct {
	onRun        func(s *KinsumerAutoscaleModuleTestSuite)
	setupMocks   func(s *KinsumerAutoscaleModuleTestSuite)
	returnsError bool
}

var kinsumerAutoscaleModuleTestCases = map[string]kinsumerAutoscaleModuleTestCase{
	"not_leader": {
		onRun: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.logger.EXPECT().WithContext(s.ctx).Return(s.logger).Once()
			s.mockLeaderElection(false, nil, false)
			s.logger.EXPECT().Info("not leading: do nothing").Once()
		},
		returnsError: false,
	},
	"leader_failed_autoscale_anyway": {
		onRun: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.logger.EXPECT().WithContext(s.ctx).Return(s.logger).Once()
			err := fmt.Errorf("unknown leader election error")
			s.mockLeaderElection(false, err, false)
			s.logger.EXPECT().Warn("will assume leader role as election failed: %s", err).Once()
			s.mockGetCurrentTaskCount(4, nil)
			s.mockGetShardCount(6, nil)
			s.mockUpdateTaskCount(6, nil)
			s.logger.EXPECT().Info("scaled task count from %d to %d", int32(4), int32(6)).Once()
		},
		returnsError: false,
	},
	"leader_failed_fatal": {
		onRun: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.logger.EXPECT().WithContext(s.ctx).Return(s.logger).Once()
			s.mockLeaderElection(false, conc.NewLeaderElectionFatalError(s.returnedError), false)
		},
		returnsError: true,
	},
	"nothing_to_do": {
		onRun: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.logger.EXPECT().WithContext(s.ctx).Return(s.logger).Once()
			s.mockLeaderElection(true, nil, false)
			s.mockGetCurrentTaskCount(4, nil)
			s.mockGetShardCount(4, nil)
		},
		returnsError: false,
	},
	"autoscale": {
		onRun: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.logger.EXPECT().WithContext(s.ctx).Return(s.logger).Once()
			s.mockLeaderElection(true, nil, false)
			s.mockGetCurrentTaskCount(6, nil)
			s.mockGetShardCount(4, nil)
			s.mockUpdateTaskCount(4, nil)
			s.logger.EXPECT().Info("scaled task count from %d to %d", int32(6), int32(4)).Once()
		},
		returnsError: false,
	},
	"get_current_task_count_failed": {
		onRun: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.logger.EXPECT().WithContext(s.ctx).Return(s.logger).Once()
			s.mockLeaderElection(true, nil, true)
			s.mockGetCurrentTaskCount(6, s.returnedError)
		},
		returnsError: true,
	},
	"get_shard_count_failed": {
		onRun: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.logger.EXPECT().WithContext(s.ctx).Return(s.logger).Once()
			s.mockLeaderElection(true, nil, true)
			s.mockGetCurrentTaskCount(6, nil)
			s.mockGetShardCount(4, s.returnedError)
		},
		returnsError: true,
	},
	"update_task_count_failed": {
		onRun: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *KinsumerAutoscaleModuleTestSuite) {
			s.logger.EXPECT().WithContext(s.ctx).Return(s.logger).Once()
			s.mockLeaderElection(true, nil, true)
			s.mockGetCurrentTaskCount(6, nil)
			s.mockGetShardCount(4, nil)
			s.mockUpdateTaskCount(4, s.returnedError)
		},
		returnsError: true,
	},
}

type KinsumerAutoscaleModuleTestSuite struct {
	suite.Suite

	ctx               context.Context
	cancel            context.CancelFunc
	logger            *logMocks.Logger
	leaderElection    *concDdbMocks.LeaderElection
	kinesisClient     *kinesisMocks.Client
	orchestrator      *mocks.KinsumerAutoscaleOrchestrator
	clock             clock.Clock
	ticker            clock.Ticker
	ecsCluster        string
	ecsService        string
	kinesisStreamName string
	returnedError     error

	settings stream.KinsumerAutoscaleModuleSettings
	module   *stream.KinsumerAutoscaleModule
}

func (s *KinsumerAutoscaleModuleTestSuite) SetupTestCase() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.logger = logMocks.NewLogger(s.T())
	s.leaderElection = concDdbMocks.NewLeaderElection(s.T())
	s.kinesisClient = kinesisMocks.NewClient(s.T())
	s.orchestrator = mocks.NewKinsumerAutoscaleOrchestrator(s.T())
	s.clock = clock.NewFakeClock()
	s.ticker = s.clock.NewTicker(time.Minute)
	s.ecsService = "my-service"
	s.ecsCluster = "gosoline-test"
	s.kinesisStreamName = "my-stream"
	s.returnedError = fmt.Errorf("some error")

	s.settings = stream.KinsumerAutoscaleModuleSettings{
		Ecs: stream.KinsumerAutoscaleModuleEcsSettings{
			Client:  "default",
			Cluster: s.ecsCluster,
			Service: s.ecsService,
		},
		Enabled: true,
		DynamoDb: stream.KinsumerAutoscaleModuleDynamoDbSettings{
			Naming: stream.KinsumerAutoscaleModuleDynamoDbNamingSettings{
				Pattern: "default",
			},
		},
		LeaderElection: "kinsumer-autoscale",
		Orchestrator:   "ecs",
		Period:         time.Minute,
	}

	var err error
	s.module = stream.NewKinsumerAutoscaleModuleWithInterfaces(
		s.logger,
		s.kinesisClient,
		s.kinesisStreamName,
		s.leaderElection,
		"leader-member-id",
		s.orchestrator,
		s.settings,
		s.ticker,
	)
	s.NoError(err)
}

func (s *KinsumerAutoscaleModuleTestSuite) TestModule() {
	for name, tc := range kinsumerAutoscaleModuleTestCases {
		s.Run(name, func() {
			var err error
			var wg sync.WaitGroup
			wg.Add(1)

			s.SetupTestCase()
			tc.setupMocks(s)

			go func() {
				defer wg.Done()
				err = s.module.Run(s.ctx)
			}()

			tc.onRun(s)
			wg.Wait()

			if tc.returnsError {
				s.ErrorContains(err, s.returnedError.Error())
			} else {
				s.NoError(err)
			}
		})
	}
}

func (s *KinsumerAutoscaleModuleTestSuite) mockLeaderElection(result bool, err error, resign bool) {
	s.leaderElection.EXPECT().IsLeader(s.ctx, "leader-member-id").Return(result, err).Once()
	if resign {
		s.leaderElection.EXPECT().Resign(s.ctx, "leader-member-id").Return(nil).Once()
	}
}

func (s *KinsumerAutoscaleModuleTestSuite) mockGetCurrentTaskCount(taskCount int32, err error) {
	s.orchestrator.EXPECT().GetCurrentTaskCount(s.ctx).Return(taskCount, err).Once()
}

func (s *KinsumerAutoscaleModuleTestSuite) mockUpdateTaskCount(taskCount int32, err error) {
	s.orchestrator.EXPECT().UpdateTaskCount(s.ctx, taskCount).Return(err).Once()
}

func (s *KinsumerAutoscaleModuleTestSuite) mockGetShardCount(shardCount int32, err error) {
	input := &kinesis.DescribeStreamSummaryInput{
		StreamName: aws.String(s.kinesisStreamName),
	}
	output := &kinesis.DescribeStreamSummaryOutput{
		StreamDescriptionSummary: &types.StreamDescriptionSummary{
			OpenShardCount: aws.Int32(shardCount),
		},
	}
	s.kinesisClient.EXPECT().DescribeStreamSummary(s.ctx, input).Return(output, err).Once()
}
