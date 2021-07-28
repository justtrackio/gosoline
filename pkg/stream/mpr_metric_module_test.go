package stream_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	cloudMocks "github.com/applike/gosoline/pkg/cloud/mocks"
	concMocks "github.com/applike/gosoline/pkg/conc/mocks"
	"github.com/applike/gosoline/pkg/log"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/metric"
	metricMocks "github.com/applike/gosoline/pkg/metric/mocks"
	"github.com/applike/gosoline/pkg/stream"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

type mprMetricModuleTestCase struct {
	onRun      func(s *MprMetricModuleTestSuite)
	setupMocks func(s *MprMetricModuleTestSuite)
}

var mprMetricModuleTestCases = map[string]mprMetricModuleTestCase{
	"not_leader": {
		onRun: func(s *MprMetricModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *MprMetricModuleTestSuite) {
			s.mockLeaderElection(false, nil)
			s.logger.On("Info", "not leading: do nothing")
		},
	},
	"leader_failed": {
		onRun: func(s *MprMetricModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *MprMetricModuleTestSuite) {
			err := fmt.Errorf("unknown leader election error")
			s.mockLeaderElection(false, err)
			s.logger.On("Warn", "will assume leader role as election failed: %s", err)
			s.mockGetMetricMessagesSent(1000, nil)
			s.mockGetMetricMessagesVisible(0, nil)
			s.mockGetMetricEcs("DesiredTaskCount", cloudwatch.StatisticMaximum, 2, nil)
			s.mockGetMetricMessagesPerRunner(499, nil)

			s.mockSuccessLogger(1000, 0, 2, 500)
			s.mockMetricWriteMessagesPerRunner(500)
		},
	},
	"happy_path": {
		onRun: func(s *MprMetricModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *MprMetricModuleTestSuite) {
			s.mockLeaderElection(true, nil)
			s.mockGetMetricMessagesSent(1000, nil)
			s.mockGetMetricMessagesVisible(0, nil)
			s.mockGetMetricEcs("DesiredTaskCount", cloudwatch.StatisticMaximum, 2, nil)
			s.mockGetMetricMessagesPerRunner(499, nil)

			s.mockSuccessLogger(1000, 0, 2, 500)
			s.mockMetricWriteMessagesPerRunner(500)
		},
	},
	"error_on_get_queue_metric_messages_sent": {
		onRun: func(s *MprMetricModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *MprMetricModuleTestSuite) {
			s.mockLeaderElection(true, nil)

			err := fmt.Errorf("unknown error")
			s.mockGetMetricMessagesSent(1000, err)
			s.logger.On("Warn", "can not calculate messages per runner: %s", mock.AnythingOfType("*fmt.wrapError")).Run(func(args mock.Arguments) {
				s.EqualError(args[1].(error), "can not get number of messages sent: can not get metric data: unknown error")
			})
		},
	},
	"error_on_get_queue_metric_messages_visisble": {
		onRun: func(s *MprMetricModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *MprMetricModuleTestSuite) {
			s.mockLeaderElection(true, nil)
			s.mockGetMetricMessagesSent(1000, nil)

			err := fmt.Errorf("unknown error")
			s.mockGetMetricMessagesVisible(1000, err)
			s.logger.On("Warn", "can not calculate messages per runner: %s", mock.AnythingOfType("*fmt.wrapError")).Run(func(args mock.Arguments) {
				s.EqualError(args[1].(error), "can not get number of messages visible: can not get metric data: unknown error")
			})
		},
	},
	"error_on_get_mpr_metric_runner_count": {
		onRun: func(s *MprMetricModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *MprMetricModuleTestSuite) {
			s.mockLeaderElection(true, nil)
			s.mockGetMetricMessagesSent(1000, nil)
			s.mockGetMetricMessagesVisible(0, nil)

			err := fmt.Errorf("unknown error")
			s.mockGetMetricEcs("DesiredTaskCount", cloudwatch.StatisticMaximum, 2, err)
			s.logger.On("Warn", "can not calculate messages per runner: %s", mock.AnythingOfType("*fmt.wrapError")).Run(func(args mock.Arguments) {
				s.EqualError(args[1].(error), "can not get runner count: can not get metric: unknown error")
			})
		},
	},
	"runner_count_zero": {
		onRun: func(s *MprMetricModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *MprMetricModuleTestSuite) {
			s.mockLeaderElection(true, nil)
			s.mockGetMetricMessagesSent(1000, nil)
			s.mockGetMetricMessagesVisible(0, nil)

			s.mockGetMetricEcs("DesiredTaskCount", cloudwatch.StatisticMaximum, 0, nil)
			s.logger.On("Warn", "can not calculate messages per runner: %s", mock.Anything).Run(func(args mock.Arguments) {
				s.EqualError(args[1].(error), "runner count is zero")
			})
		},
	},
	"error_on_get_mpr_metric_messages_per_runner": {
		onRun: func(s *MprMetricModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *MprMetricModuleTestSuite) {
			s.mockLeaderElection(true, nil)
			s.mockGetMetricMessagesSent(1000, nil)
			s.mockGetMetricMessagesVisible(0, nil)
			s.mockGetMetricEcs("DesiredTaskCount", cloudwatch.StatisticMaximum, 2, nil)

			err := fmt.Errorf("unknown error")
			s.mockGetMetricMessagesPerRunner(500, err)
			s.logger.On("Warn", "can not get current messages per runner metric: defaulting to 0")

			s.mockSuccessLogger(1000, 0, 2, 500)
			s.mockMetricWriteMessagesPerRunner(500)
		},
	},
	"max_mpr_crossed": {
		onRun: func(s *MprMetricModuleTestSuite) {
			s.cancel()
		},
		setupMocks: func(s *MprMetricModuleTestSuite) {
			s.mockLeaderElection(true, nil)
			s.mockGetMetricMessagesSent(2000, nil)
			s.mockGetMetricMessagesVisible(0, nil)
			s.mockGetMetricEcs("DesiredTaskCount", cloudwatch.StatisticMaximum, 2, nil)
			s.mockGetMetricMessagesPerRunner(499, nil)

			s.logger.On("Warn", "newMpr of %f is higher than configured maxMpr of %f: falling back to max", 1000.0, 998.0)

			s.mockSuccessLogger(2000, 0, 2, 998)
			s.mockMetricWriteMessagesPerRunner(998)
		},
	},
}

type MprMetricModuleTestSuite struct {
	suite.Suite

	ctx            context.Context
	cancel         context.CancelFunc
	logger         *logMocks.Logger
	leaderElection *concMocks.LeaderElection
	cwClient       *cloudMocks.CloudWatchAPI
	metricWriter   *metricMocks.Writer
	clock          clock.Clock
	ticker         clock.Ticker

	settings *stream.MessagesPerRunnerMetricWriterSettings
	writer   *stream.MessagesPerRunnerMetricWriter
}

func (s *MprMetricModuleTestSuite) SetupTestCase() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.logger = new(logMocks.Logger)
	s.leaderElection = new(concMocks.LeaderElection)
	s.cwClient = new(cloudMocks.CloudWatchAPI)
	s.metricWriter = new(metricMocks.Writer)
	s.clock = clock.NewFakeClock()
	s.ticker = clock.NewFakeTicker()

	s.settings = &stream.MessagesPerRunnerMetricWriterSettings{
		QueueNames:         []string{"queueName"},
		UpdatePeriod:       time.Minute,
		MaxIncreasePercent: 200,
		MaxIncreasePeriod:  time.Minute * 5,
		AppId: cfg.AppId{
			Project:     "gosoline",
			Environment: "test",
			Family:      "stream",
			Application: "mprMetric",
		},
		MemberId: "e7c6003c-66df-11eb-9bdf-af0dafba2813",
	}

	var err error
	s.writer, err = stream.NewMessagesPerRunnerMetricWriterWithInterfaces(s.logger, s.leaderElection, s.cwClient, s.metricWriter, s.clock, s.ticker, s.settings)
	s.NoError(err)
}

func (s *MprMetricModuleTestSuite) TestModule() {
	for name, tc := range mprMetricModuleTestCases {
		s.Run(name, func() {
			var err error

			wg := sync.WaitGroup{}
			wg.Add(1)

			s.SetupTestCase()
			tc.setupMocks(s)

			go func() {
				defer wg.Done()
				err = s.writer.Run(s.ctx)
			}()

			tc.onRun(s)
			wg.Wait()

			s.NoError(err)
		})
	}
}

func (s *MprMetricModuleTestSuite) mockMetricWriteMessagesPerRunner(value float64) {
	s.metricWriter.On("WriteOne", &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  s.clock.Now(),
		MetricName: "StreamMprMessagesPerRunner",
		Value:      value,
		Unit:       metric.UnitCountAverage,
	})
}

func (s *MprMetricModuleTestSuite) mockLeaderElection(result bool, err error) {
	s.leaderElection.On("IsLeader", s.ctx, "e7c6003c-66df-11eb-9bdf-af0dafba2813").Return(result, err)
}

func (s *MprMetricModuleTestSuite) mockGetMetricMessagesSent(value float64, err error) {
	s.mockGetMetricMessages("NumberOfMessagesSent", cloudwatch.StatisticSum, value, err)
}

func (s *MprMetricModuleTestSuite) mockGetMetricMessagesVisible(value float64, err error) {
	s.mockGetMetricMessages("ApproximateNumberOfMessagesVisible", cloudwatch.StatisticMaximum, value, err)
}

func (s *MprMetricModuleTestSuite) mockGetMetricMessages(metric string, stat string, value float64, err error) {
	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(s.clock.Now().Add(-1 * s.settings.UpdatePeriod * 5)),
		EndTime:   aws.Time(s.clock.Now().Add(-1 * s.settings.UpdatePeriod)),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("m0"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String("AWS/SQS"),
						MetricName: aws.String(metric),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("QueueName"),
								Value: aws.String(s.settings.QueueNames[0]),
							},
						},
					},
					Period: aws.Int64(60),
					Stat:   aws.String(stat),
					Unit:   aws.String(cloudwatch.StandardUnitCount),
				},
			},
		},
	}
	output := &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []*cloudwatch.MetricDataResult{
			{
				Values: []*float64{
					aws.Float64(value),
				},
			},
		},
	}

	s.cwClient.On("GetMetricData", input).Return(output, err)
}

func (s *MprMetricModuleTestSuite) mockGetMetricMessagesPerRunner(value float64, err error) {
	s.mockGetMetricMpr("StreamMprMessagesPerRunner", cloudwatch.StatisticAverage, value, err)
}

func (s *MprMetricModuleTestSuite) mockGetMetricMpr(metric string, stat string, value float64, err error) {
	input := &cloudwatch.GetMetricDataInput{
		StartTime:     aws.Time(s.clock.Now().Add(-1 * s.settings.UpdatePeriod * 5)),
		EndTime:       aws.Time(s.clock.Now().Add(-1 * s.settings.UpdatePeriod)),
		MaxDatapoints: aws.Int64(1),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String("gosoline/test/stream/mprMetric"),
						MetricName: aws.String(metric),
					},
					Period: aws.Int64(60),
					Stat:   aws.String(stat),
					Unit:   aws.String(cloudwatch.StandardUnitCount),
				},
			},
		},
	}
	output := &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []*cloudwatch.MetricDataResult{
			{
				Values: []*float64{
					aws.Float64(value),
				},
			},
		},
	}

	s.cwClient.On("GetMetricData", input).Return(output, err)
}

func (s *MprMetricModuleTestSuite) mockGetMetricEcs(metric string, stat string, value float64, err error) {
	input := &cloudwatch.GetMetricDataInput{
		StartTime:     aws.Time(s.clock.Now().Add(-1 * s.settings.UpdatePeriod * 5)),
		EndTime:       aws.Time(s.clock.Now().Add(-1 * s.settings.UpdatePeriod)),
		MaxDatapoints: aws.Int64(1),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String("ECS/ContainerInsights"),
						MetricName: aws.String(metric),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("ClusterName"),
								Value: aws.String("gosoline-test-stream"),
							},
							{
								Name:  aws.String("ServiceName"),
								Value: aws.String("mprMetric"),
							},
						},
					},
					Period: aws.Int64(60),
					Stat:   aws.String(stat),
					Unit:   aws.String(cloudwatch.StandardUnitCount),
				},
			},
		},
	}
	output := &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []*cloudwatch.MetricDataResult{
			{
				Values: []*float64{
					aws.Float64(value),
				},
			},
		},
	}

	s.cwClient.On("GetMetricData", input).Return(output, err)
}

func (s *MprMetricModuleTestSuite) mockSuccessLogger(sent, visible, runnerCount, mpr float64) {
	s.logger.On("WithFields", log.Fields{
		"messagesSent":      sent,
		"messagesVisible":   visible,
		"runnerCount":       runnerCount,
		"messagesPerRunner": mpr,
	}).Return(s.logger)
	s.logger.On("Info", "%f messages per runner", mpr)
}

func TestMprMetricModuleTestSuite(t *testing.T) {
	suite.Run(t, new(MprMetricModuleTestSuite))
}
