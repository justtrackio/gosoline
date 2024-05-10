package calculator_test

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
	"github.com/justtrackio/gosoline/pkg/log"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/metric/calculator"
	"github.com/stretchr/testify/suite"
)

func TestPerRunnerMetricHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(PerRunnerMetricHandlerTestSuite))
}

type PerRunnerMetricHandlerTestSuite struct {
	suite.Suite

	ctx             context.Context
	logger          *logMocks.Logger
	clock           clock.Clock
	cwClient        *cloudwatchMocks.Client
	handlerSettings *calculator.PerRunnerMetricSettings
	handler         calculator.PerRunnerMetricHandler
}

func (s *PerRunnerMetricHandlerTestSuite) SetupTest() {
	s.ctx = s.T().Context()
	s.logger = new(logMocks.Logger)
	s.clock = clock.NewFakeClock()
	s.cwClient = new(cloudwatchMocks.Client)

	calculatorSettings := &calculator.CalculatorSettings{
		Ecs: calculator.EcsSettings{
			Cluster: "gosoline-test-metric",
			Service: "grp/calculator",
		},
		Period:              time.Minute,
		CloudWatchNamespace: "gosoline/test/metric/calculator",
	}

	s.handlerSettings = &calculator.PerRunnerMetricSettings{
		MaxIncreasePercent: 200,
		MaxIncreasePeriod:  time.Minute,
		Period:             time.Minute,
		TargetValue:        4,
	}

	s.handler = calculator.NewPerRunnerMetricHandlerWithInterfaces(s.logger, s.clock, s.cwClient, calculatorSettings)
}

func (s *PerRunnerMetricHandlerTestSuite) TestEcsMetricError() {
	expectedErr := fmt.Errorf("this is a cloudwatch error")
	s.mockGetRunnerCountMetricEcs(2, expectedErr)

	_, actualErr := s.handler.CalculatePerRunnerMetrics(s.ctx, "Requests", 100, s.handlerSettings)
	s.EqualError(actualErr, "can not get runner count: can not get metric: this is a cloudwatch error")
}

func (s *PerRunnerMetricHandlerTestSuite) TestZeroRunnerCount() {
	s.mockGetRunnerCountMetricEcs(0, nil)

	_, actualErr := s.handler.CalculatePerRunnerMetrics(s.ctx, "Requests", 100, s.handlerSettings)
	s.EqualError(actualErr, "runner count is zero")
}

func (s *PerRunnerMetricHandlerTestSuite) TestGetPreviousMetricError() {
	s.mockGetRunnerCountMetricEcs(2, nil)
	s.mockGetPreviousMetric(0, fmt.Errorf("previous metric error"))

	s.logger.EXPECT().Warn("can not get current %s metric per runner metric: %s, defaulting to 0", "PerRunnerRequests", "can not get metric: previous metric error")
	s.mockSuccessLogger(100, 50, 50, 2)

	expectedDatum := s.getExpectedDatum(50)

	actualDatum, actualErr := s.handler.CalculatePerRunnerMetrics(s.ctx, "Requests", 100, s.handlerSettings)
	s.NoError(actualErr)
	s.Equal(expectedDatum, actualDatum)
}

func (s *PerRunnerMetricHandlerTestSuite) TestMaxIncreaseExeeded() {
	s.mockGetRunnerCountMetricEcs(2, nil)
	s.mockGetPreviousMetric(4, nil)

	s.logger.EXPECT().Warn("newPrm of %f is higher than configured maxPrm of %f: falling back to max", float64(50), float64(8))
	s.mockSuccessLogger(100, 4, 8, 2)

	expectedDatum := s.getExpectedDatum(8)

	actualDatum, actualErr := s.handler.CalculatePerRunnerMetrics(s.ctx, "Requests", 100, s.handlerSettings)
	s.NoError(actualErr)
	s.Equal(expectedDatum, actualDatum)
}

func (s *PerRunnerMetricHandlerTestSuite) TestHappyPathNoChange() {
	s.mockGetRunnerCountMetricEcs(2, nil)
	s.mockGetPreviousMetric(4, nil)
	s.mockSuccessLogger(8, 4, 4, 2)

	expectedDatum := s.getExpectedDatum(4)

	actualDatum, actualErr := s.handler.CalculatePerRunnerMetrics(s.ctx, "Requests", 8, s.handlerSettings)
	s.NoError(actualErr)
	s.Equal(expectedDatum, actualDatum)
}

func (s *PerRunnerMetricHandlerTestSuite) TestHappyPathWithChange() {
	s.mockGetRunnerCountMetricEcs(2, nil)
	s.mockGetPreviousMetric(4, nil)
	s.mockSuccessLogger(16, 4, 8, 2)

	expectedDatum := s.getExpectedDatum(8)

	actualDatum, actualErr := s.handler.CalculatePerRunnerMetrics(s.ctx, "Requests", 16, s.handlerSettings)
	s.NoError(actualErr)
	s.Equal(expectedDatum, actualDatum)
}

func (s *PerRunnerMetricHandlerTestSuite) mockGetPreviousMetric(value float64, err error) {
	input := &cloudwatch.GetMetricDataInput{
		StartTime:     aws.Time(s.clock.Now().Add(-1 * (s.handlerSettings.MaxIncreasePeriod + s.handlerSettings.Period))),
		EndTime:       aws.Time(s.clock.Now().Add(-1 * s.handlerSettings.Period)),
		MaxDatapoints: aws.Int32(1),
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String("gosoline/test/metric/calculator"),
						MetricName: aws.String("PerRunnerRequests"),
					},
					Period: aws.Int32(60),
					Stat:   aws.String(string(types.StatisticAverage)),
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

func (s *PerRunnerMetricHandlerTestSuite) mockGetRunnerCountMetricEcs(value float64, err error) {
	input := &cloudwatch.GetMetricDataInput{
		StartTime:     aws.Time(s.clock.Now().Add(-1 * s.handlerSettings.Period * 5)),
		EndTime:       aws.Time(s.clock.Now().Add(-1 * s.handlerSettings.Period)),
		MaxDatapoints: aws.Int32(1),
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String("ECS/ContainerInsights"),
						MetricName: aws.String("DesiredTaskCount"),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String("ClusterName"),
								Value: aws.String("gosoline-test-metric"),
							},
							{
								Name:  aws.String("ServiceName"),
								Value: aws.String("grp/calculator"),
							},
						},
					},
					Period: aws.Int32(60),
					Stat:   aws.String(string(types.StatisticMaximum)),
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

func (s *PerRunnerMetricHandlerTestSuite) mockSuccessLogger(currentValue, currentPrm, newPrm, runnerCount float64) {
	s.logger.EXPECT().WithFields(log.Fields{
		"currentValue": currentValue,
		"currentPrm":   currentPrm,
		"newPrm":       newPrm,
		"runnerCount":  runnerCount,
	}).Return(s.logger)

	s.logger.EXPECT().Info("%s evaluated to %f", "PerRunnerRequests", newPrm)
}

func (s *PerRunnerMetricHandlerTestSuite) getExpectedDatum(value float64) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: "PerRunnerRequests",
		Value:      value,
		Unit:       metric.UnitCountAverage,
	}
}
