package stream_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	cloudwatchMocks "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/metric/calculator"
	"github.com/justtrackio/gosoline/pkg/metric/calculator/mocks"
	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

func TestMessagesPerRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(MessagesPerRunnerTestSuite))
}

type MessagesPerRunnerTestSuite struct {
	suite.Suite

	ctx             context.Context
	logger          *logMocks.Logger
	clock           clock.Clock
	cwClient        *cloudwatchMocks.Client
	baseHandler     *mocks.PerRunnerMetricHandler
	handlerSettings *calculator.PerRunnerMetricSettings
	handler         calculator.Handler
}

func (s *MessagesPerRunnerTestSuite) SetupTest() {
	s.ctx = s.T().Context()
	s.logger = new(logMocks.Logger)
	s.clock = clock.NewFakeClock()
	s.cwClient = new(cloudwatchMocks.Client)
	s.baseHandler = new(mocks.PerRunnerMetricHandler)

	calculatorSettings := &calculator.CalculatorSettings{
		Ecs: calculator.EcsSettings{
			Cluster: "gosoline-test-metric",
			Service: "grp/calculator",
		},
		LeaderElection:      "metric_calculator",
		Period:              time.Minute,
		CloudWatchNamespace: "gosoline/test/httpserver/demo",
	}

	s.handlerSettings = &calculator.PerRunnerMetricSettings{
		MaxIncreasePercent: 200,
		MaxIncreasePeriod:  time.Minute,
		Period:             time.Minute,
		TargetValue:        4,
	}

	queueNames := []string{"default"}
	s.handler = stream.NewMessagesPerRunnerHandlerWithInterfaces(s.logger, s.clock, s.cwClient, s.baseHandler, calculatorSettings, s.handlerSettings, queueNames)
}

func (s *MessagesPerRunnerTestSuite) TearDownTest() {
	s.logger.AssertExpectations(s.T())
	s.cwClient.AssertExpectations(s.T())
	s.baseHandler.AssertExpectations(s.T())
}

func (s *MessagesPerRunnerTestSuite) TestGetRequestsMetricsError() {
	s.mockGetSqsMetrics("NumberOfMessagesSent", types.StatisticSum, 0, fmt.Errorf("some cloudwatch error"))

	_, actualError := s.handler.GetMetrics(s.ctx)
	s.EqualError(actualError, "can not get number of messages: can not get number of messages sent: can not get metric data: some cloudwatch error")
}

func (s *MessagesPerRunnerTestSuite) TestCalculatePerRunnerMetricsError() {
	s.mockGetSqsMetrics("NumberOfMessagesSent", types.StatisticSum, 100, nil)
	s.mockGetSqsMetrics("ApproximateNumberOfMessagesVisible", types.StatisticMaximum, 50, nil)
	s.mockBaseHandler(150, nil, fmt.Errorf("base handler error"))
	s.logger.EXPECT().Warn(matcher.Context, "can not calculate metrics per runner for handler: can not calculate httpserver per runner metrics: %s: %T", "base handler error", mock.Anything)

	expectedSmaDatum := &metric.Datum{
		Priority:   metric.PriorityLow,
		Timestamp:  time.Time{},
		MetricName: "StreamMessagesAvailable",
		Value:      50,
		Unit:       metric.UnitCount,
	}

	expectedSmsDatum := &metric.Datum{
		Priority:   metric.PriorityLow,
		Timestamp:  time.Time{},
		MetricName: "StreamMessagesSent",
		Value:      100,
		Unit:       metric.UnitCount,
	}

	data, err := s.handler.GetMetrics(s.ctx)
	s.Nil(err, "err should be nil for mpr GetMetrics upon calculation error, as we still want to use the stream metrics")
	s.Equal(metric.Data{
		expectedSmaDatum,
		expectedSmsDatum,
	}, data)
}

func (s *MessagesPerRunnerTestSuite) TestSuccess() {
	expectedPrsmDatum := &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Time{},
		MetricName: "PerRunnerStreamMessages",
		Value:      50,
		Unit:       metric.UnitCountAverage,
	}

	expectedSmaDatum := &metric.Datum{
		Priority:   metric.PriorityLow,
		Timestamp:  time.Time{},
		MetricName: "StreamMessagesAvailable",
		Value:      50,
		Unit:       metric.UnitCount,
	}

	expectedSmsDatum := &metric.Datum{
		Priority:   metric.PriorityLow,
		Timestamp:  time.Time{},
		MetricName: "StreamMessagesSent",
		Value:      100,
		Unit:       metric.UnitCount,
	}

	s.mockGetSqsMetrics("NumberOfMessagesSent", types.StatisticSum, 100, nil)
	s.mockGetSqsMetrics("ApproximateNumberOfMessagesVisible", types.StatisticMaximum, 50, nil)
	s.mockBaseHandler(150, expectedPrsmDatum, nil)

	actualData, actualError := s.handler.GetMetrics(s.ctx)
	s.NoError(actualError)
	s.Equal(metric.Data{
		expectedPrsmDatum,
		expectedSmaDatum,
		expectedSmsDatum,
	}, actualData)
}

func (s *MessagesPerRunnerTestSuite) mockGetSqsMetrics(metricName string, typ types.Statistic, value float64, err error) {
	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(s.clock.Now().Add(-1 * s.handlerSettings.Period * 5)),
		EndTime:   aws.Time(s.clock.Now().Add(-1 * s.handlerSettings.Period)),
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("m_0"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String("AWS/SQS"),
						MetricName: aws.String(metricName),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String("QueueName"),
								Value: aws.String("default"),
							},
						},
					},
					Period: aws.Int32(60),
					Stat:   aws.String(string(typ)),
					Unit:   types.StandardUnitCount,
				},
			},
		},
	}
	output := &cloudwatch.GetMetricDataOutput{
		MetricDataResults: []types.MetricDataResult{
			{
				Values: []float64{
					value,
				},
			},
		},
	}

	s.cwClient.EXPECT().GetMetricData(s.ctx, input).Return(output, err)
}

func (s *MessagesPerRunnerTestSuite) mockBaseHandler(value float64, datum *metric.Datum, err error) {
	s.baseHandler.EXPECT().
		CalculatePerRunnerMetrics(s.ctx, stream.PerRunnerMetricName, value, s.handlerSettings).
		Return(datum, err)
}
