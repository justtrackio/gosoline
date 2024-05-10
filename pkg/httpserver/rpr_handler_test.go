package httpserver_test

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
	"github.com/justtrackio/gosoline/pkg/httpserver"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/metric/calculator"
	"github.com/justtrackio/gosoline/pkg/metric/calculator/mocks"
	"github.com/stretchr/testify/suite"
)

func TestRequestsPerRunnerTestSuite(t *testing.T) {
	suite.Run(t, new(RequestsPerRunnerTestSuite))
}

type RequestsPerRunnerTestSuite struct {
	suite.Suite

	ctx             context.Context
	logger          *logMocks.Logger
	clock           clock.Clock
	cwClient        *cloudwatchMocks.Client
	baseHandler     *mocks.PerRunnerMetricHandler
	handlerSettings *calculator.PerRunnerMetricSettings
	handler         calculator.Handler
}

func (s *RequestsPerRunnerTestSuite) SetupTest() {
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

	serverNames := []string{"default"}
	s.handler = httpserver.NewRequestsPerRunnerHandlerWithInterfaces(s.clock, s.cwClient, s.baseHandler, calculatorSettings, s.handlerSettings, serverNames)
}

func (s *RequestsPerRunnerTestSuite) TearDownTest() {
	s.logger.AssertExpectations(s.T())
	s.cwClient.AssertExpectations(s.T())
	s.baseHandler.AssertExpectations(s.T())
}

func (s *RequestsPerRunnerTestSuite) TestGetRequestsMetricsError() {
	s.mockGetRequestsMetrics(0, fmt.Errorf("some cloudwatch error"))

	_, actualError := s.handler.GetMetrics(s.ctx)
	s.EqualError(actualError, "can not get number of requests received: can not get metric: some cloudwatch error")
}

func (s *RequestsPerRunnerTestSuite) TestCalculatePerRunnerMetricsError() {
	s.mockGetRequestsMetrics(100, nil)
	s.mockBaseHandler(100, nil, fmt.Errorf("base handler error"))

	_, actualError := s.handler.GetMetrics(s.ctx)
	s.EqualError(actualError, "can not calculate httpserver per runner metrics: base handler error")
}

func (s *RequestsPerRunnerTestSuite) TestSuccess() {
	expectedDatum := &metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  time.Time{},
		MetricName: "PerRunnerHttpServerRequests",
		Value:      50,
		Unit:       metric.UnitCountAverage,
	}

	s.mockGetRequestsMetrics(100, nil)
	s.mockBaseHandler(100, expectedDatum, nil)

	actualData, actualError := s.handler.GetMetrics(s.ctx)
	s.NoError(actualError)
	s.Equal(metric.Data{expectedDatum}, actualData)
}

func (s *RequestsPerRunnerTestSuite) mockGetRequestsMetrics(value float64, err error) {
	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(s.clock.Now().Add(-1 * s.handlerSettings.Period * 5)),
		EndTime:   aws.Time(s.clock.Now().Add(-1 * s.handlerSettings.Period)),
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("m_default"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String("gosoline/test/httpserver/demo"),
						MetricName: aws.String("HttpRequestCount"),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String("ServerName"),
								Value: aws.String("default"),
							},
						},
					},
					Period: aws.Int32(60),
					Stat:   aws.String(string(types.StatisticSum)),
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

func (s *RequestsPerRunnerTestSuite) mockBaseHandler(value float64, datum *metric.Datum, err error) {
	s.baseHandler.EXPECT().
		CalculatePerRunnerMetrics(s.ctx, httpserver.PerRunnerMetricName, value, s.handlerSettings).
		Return(datum, err)
}
