package calculator_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	cloudwatchMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch/mocks"
	concDdbMocks "github.com/justtrackio/gosoline/pkg/conc/ddb/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/metric/calculator"
	"github.com/justtrackio/gosoline/pkg/metric/calculator/mocks"
	metricMocks "github.com/justtrackio/gosoline/pkg/metric/mocks"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/suite"
)

func TestCalculatorModuleTestSuite(t *testing.T) {
	suite.Run(t, new(CalculatorModuleTestSuite))
}

var dummyMetrics = metric.Data{
	{
		Priority:   metric.PriorityHigh,
		Timestamp:  clock.NewFakeClock().Now(),
		MetricName: "test-handler-metric",
		Dimensions: nil,
		Value:      13.37,
		Unit:       metric.UnitCountAverage,
	},
}

type calculatorModuleTestCase struct {
	onRun      func(s *CalculatorModuleTestSuite)
	setupMocks func(s *CalculatorModuleTestSuite)
}

var calculatorModuleTestCases = map[string]calculatorModuleTestCase{
	"not_leader": {
		onRun: func(s *CalculatorModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *CalculatorModuleTestSuite) {
			s.mockLeaderElection(false, nil)
			s.logger.EXPECT().Info(matcher.Context, "not leading: do nothing")
		},
	},
	"leader_failed": {
		onRun: func(s *CalculatorModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *CalculatorModuleTestSuite) {
			err := fmt.Errorf("unknown leader election error")
			s.mockLeaderElection(false, err)
			s.logger.EXPECT().Warn(matcher.Context, "will assume leader role as election failed: %s", err)
			s.mockHandler(dummyMetrics, nil)
			s.metricWriter.EXPECT().Write(matcher.Context, dummyMetrics)
		},
	},
	"happy_path": {
		onRun: func(s *CalculatorModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *CalculatorModuleTestSuite) {
			s.mockLeaderElection(true, nil)
			s.mockHandler(dummyMetrics, nil)
			s.metricWriter.EXPECT().Write(matcher.Context, dummyMetrics)
		},
	},
	"handler_failed": {
		onRun: func(s *CalculatorModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *CalculatorModuleTestSuite) {
			err := fmt.Errorf("handler failed")
			var allMetrics metric.Data

			s.mockLeaderElection(true, nil)
			s.mockHandler(nil, err)
			s.logger.EXPECT().Warn(matcher.Context, "can not calculate metrics per runner for handler %s: %s", "requests", err)
			s.metricWriter.EXPECT().Write(matcher.Context, allMetrics)
		},
	},
}

type CalculatorModuleTestSuite struct {
	suite.Suite

	ctx            context.Context
	cancel         context.CancelFunc
	logger         *logMocks.Logger
	leaderElection *concDdbMocks.LeaderElection
	cwClient       *cloudwatchMocks.Client
	handler        *mocks.Handler
	metricWriter   *metricMocks.Writer
	clock          clock.Clock
	ticker         clock.Ticker

	settings *calculator.CalculatorSettings
	module   *calculator.CalculatorModule
}

func (s *CalculatorModuleTestSuite) SetupTestCase() {
	s.ctx, s.cancel = context.WithCancel(s.T().Context())

	s.logger = new(logMocks.Logger)
	s.leaderElection = new(concDdbMocks.LeaderElection)
	s.cwClient = new(cloudwatchMocks.Client)
	s.handler = new(mocks.Handler)
	s.metricWriter = new(metricMocks.Writer)
	s.clock = clock.NewFakeClock()
	s.ticker = s.clock.NewTicker(time.Minute)

	s.settings = &calculator.CalculatorSettings{
		Cloudwatch: calculator.CloudWatchSettings{
			Client: "default",
		},
		DynamoDb: calculator.DynamoDbSettings{
			Naming: calculator.DynamoDbNamingSettings{
				TablePattern: "{app.tags.project}-{app.env}-{app.tags.family}-{app.tags.group}",
			},
		},
		Ecs: calculator.EcsSettings{
			Cluster: "gosoline-test-metric",
			Service: "grp/calculator",
		},
		LeaderElection:      "metric_calculator",
		Period:              time.Minute,
		CloudWatchNamespace: "gosoline/test/metric/calculator",
	}

	handlers := map[string]calculator.Handler{
		"requests": s.handler,
	}

	var err error
	s.module = calculator.NewCalculatorModuleWithInterfaces(s.logger, s.leaderElection, s.cwClient, s.metricWriter, s.ticker, handlers, "leader-member-id", s.settings)
	s.NoError(err)
}

func (s *CalculatorModuleTestSuite) TestModule() {
	for name, tc := range calculatorModuleTestCases {
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

			s.NoError(err)
		})
	}
}

func (s *CalculatorModuleTestSuite) mockLeaderElection(result bool, err error) {
	s.leaderElection.EXPECT().IsLeader(s.ctx, "leader-member-id").Return(result, err)
}

func (s *CalculatorModuleTestSuite) mockHandler(result metric.Data, err error) {
	s.handler.EXPECT().GetMetrics(s.ctx).Return(result, err)
}
