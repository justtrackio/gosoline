package calculator

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoCloudwatch "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

func getPerRunnerMetricName(name string) string {
	return fmt.Sprintf("PerRunner%s", name)
}

type PerRunnerMetricSettings struct {
	Enabled            bool          `cfg:"enabled" default:"true"`
	MaxIncreasePercent float64       `cfg:"max_increase_percent" default:"200"`
	MaxIncreasePeriod  time.Duration `cfg:"max_increase_period" default:"5m"`
	Period             time.Duration `cfg:"period" default:"1m"`
	TargetValue        float64       `cfg:"target_value" default:"0"`
}

//go:generate mockery --name PerRunnerMetricHandler
type PerRunnerMetricHandler interface {
	CalculatePerRunnerMetrics(ctx context.Context, name string, currentValue float64, settings *PerRunnerMetricSettings) (*metric.Datum, error)
}

type perRunnerMetricHandler struct {
	logger             log.Logger
	clock              clock.Clock
	cwClient           gosoCloudwatch.Client
	calculatorSettings *CalculatorSettings
}

func NewPerRunnerMetricHandlerWithInterfaces(logger log.Logger, clock clock.Clock, cwClient gosoCloudwatch.Client, calculatorSettings *CalculatorSettings) *perRunnerMetricHandler {
	return &perRunnerMetricHandler{
		logger:             logger,
		clock:              clock,
		cwClient:           cwClient,
		calculatorSettings: calculatorSettings,
	}
}

func (h *perRunnerMetricHandler) CalculatePerRunnerMetrics(ctx context.Context, name string, currentValue float64, settings *PerRunnerMetricSettings) (*metric.Datum, error) {
	var err error
	var runnerCount, currentPrm, newPrm, maxPrm float64
	metricName := getPerRunnerMetricName(name)

	if runnerCount, err = h.getEcsMetric(ctx, "DesiredTaskCount", types.StatisticMaximum, settings.Period); err != nil {
		return nil, fmt.Errorf("can not get runner count: %w", err)
	}

	if runnerCount == 0 {
		return nil, fmt.Errorf("runner count is zero")
	}

	if currentPrm, err = h.getPreviousMetric(ctx, metricName, types.StatisticAverage, settings); err != nil {
		h.logger.Warn("can not get current %s metric per runner metric: %s, defaulting to 0", metricName, err.Error())
		currentPrm = 0
	}

	newPrm = currentValue / runnerCount

	if currentPrm == 0 {
		currentPrm = newPrm
	}

	maxPrm = currentPrm * (settings.MaxIncreasePercent / 100)

	if currentPrm < settings.TargetValue {
		maxPrm = settings.TargetValue * (settings.MaxIncreasePercent / 100)
	}

	if newPrm > maxPrm {
		h.logger.Warn("newPrm of %f is higher than configured maxPrm of %f: falling back to max", newPrm, maxPrm)
		newPrm = maxPrm
	}

	h.logger.WithFields(log.Fields{
		"currentPrm":   currentPrm,
		"currentValue": currentValue,
		"newPrm":       newPrm,
		"runnerCount":  runnerCount,
	}).Info("%s evaluated to %f", metricName, newPrm)

	datum := &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricName,
		Unit:       metric.UnitCountAverage,
		Value:      newPrm,
	}

	return datum, nil
}

func (h *perRunnerMetricHandler) getPreviousMetric(ctx context.Context, name string, stat types.Statistic, settings *PerRunnerMetricSettings) (float64, error) {
	startTime := h.clock.Now().Add(-1 * (settings.MaxIncreasePeriod + settings.Period))
	endTime := h.clock.Now().Add(-1 * settings.Period)
	periodSeconds := int32(settings.Period.Seconds())

	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String(h.calculatorSettings.CloudWatchNamespace),
						MetricName: aws.String(name),
					},
					Period: aws.Int32(periodSeconds),
					Stat:   aws.String(string(stat)),
					Unit:   types.StandardUnitCount,
				},
			},
		},
		MaxDatapoints: aws.Int32(1),
	}

	out, err := h.cwClient.GetMetricData(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("can not get metric: %w", err)
	}

	if len(out.MetricDataResults) == 0 {
		return 0, fmt.Errorf("no metric results")
	}

	if len(out.MetricDataResults[0].Values) == 0 {
		return 0, fmt.Errorf("no metric values")
	}

	value := out.MetricDataResults[0].Values[0]

	return value, nil
}

func (h *perRunnerMetricHandler) getEcsMetric(ctx context.Context, name string, stat types.Statistic, period time.Duration) (float64, error) {
	clusterName := h.calculatorSettings.Ecs.Cluster
	serviceName := h.calculatorSettings.Ecs.Service

	startTime := h.clock.Now().Add(-1 * period * 5)
	endTime := h.clock.Now().Add(-1 * period)
	periodSeconds := int32(period.Seconds())

	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String("ECS/ContainerInsights"),
						MetricName: aws.String(name),
						Dimensions: []types.Dimension{
							{
								Name:  aws.String("ClusterName"),
								Value: aws.String(clusterName),
							},
							{
								Name:  aws.String("ServiceName"),
								Value: aws.String(serviceName),
							},
						},
					},
					Period: aws.Int32(periodSeconds),
					Stat:   aws.String(string(stat)),
					Unit:   types.StandardUnitCount,
				},
			},
		},
		MaxDatapoints: aws.Int32(1),
	}

	out, err := h.cwClient.GetMetricData(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("can not get metric: %w", err)
	}

	if len(out.MetricDataResults) == 0 {
		return 0, fmt.Errorf("no metric results")
	}

	if len(out.MetricDataResults[0].Values) == 0 {
		return 0, fmt.Errorf("no metric values")
	}

	value := out.MetricDataResults[0].Values[0]

	return value, nil
}
