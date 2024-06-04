package httpserver

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoCloudwatch "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	metricsPerRunner "github.com/justtrackio/gosoline/pkg/metric/metrics_per_runner"
	"golang.org/x/exp/maps"
)

func init() {
	metricsPerRunner.RegisterHandler("httpserver", &rprHandler{})
}

type rprHandler struct {
	settings    metricsPerRunner.HandlerSettings
	serverNames []string
	cwClient    gosoCloudwatch.Client
	cwNamespace string
	clock       clock.Clock
}

func getHttpServerNames(config cfg.Config) []string {
	return maps.Keys(config.GetStringMap("httpserver"))
}

func (h *rprHandler) IsEnabled(config cfg.Config) bool {
	return funk.Any(getHttpServerNames(config), func(serverName string) bool {
		return requestsPerRunnerIsEnabled(config, serverName)
	})
}

func (h *rprHandler) Init(ctx context.Context, config cfg.Config, logger log.Logger, cwNamespace string) (*metricsPerRunner.HandlerSettings, error) {
	var err error

	h.settings = readRequestsPerRunnerMetricHandlerSettings(config)
	h.serverNames = funk.Filter(getHttpServerNames(config), func(serverName string) bool {
		return requestsPerRunnerIsEnabled(config, serverName)
	})

	h.clock = clock.Provider
	h.cwNamespace = cwNamespace

	if h.cwClient, err = gosoCloudwatch.ProvideClient(ctx, config, logger, "default"); err != nil {
		return nil, fmt.Errorf("can not create cloudwatch client: %w", err)
	}

	return &h.settings, nil
}

func (h *rprHandler) GetMetricSum(ctx context.Context) (float64, error) {
	var err error
	var requests float64

	if requests, err = h.getRequestsMetrics(ctx, MetricHttpRequestCount, types.StatisticSum); err != nil {
		return 0, fmt.Errorf("can not get number of messages sent: %w", err)
	}

	return requests, nil
}

func (h *rprHandler) getRequestsMetrics(ctx context.Context, metric string, stat types.Statistic) (float64, error) {
	startTime := h.clock.Now().Add(-1 * h.settings.Period * 5)
	endTime := h.clock.Now().Add(-1 * h.settings.Period)
	period := int32(h.settings.Period.Seconds())
	queries := make([]types.MetricDataQuery, len(h.serverNames))

	for i, serverName := range h.serverNames {
		queries[i] = types.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m%d", i)),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  aws.String(h.cwNamespace),
					MetricName: aws.String(metric),
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("ServerName"),
							Value: aws.String(serverName),
						},
					},
				},
				Period: aws.Int32(period),
				Stat:   aws.String(string(stat)),
				Unit:   types.StandardUnitCount,
			},
		}
	}

	input := &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		MetricDataQueries: queries,
	}

	out, err := h.cwClient.GetMetricData(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("can not get metric data: %w", err)
	}

	value := 0.0
	for _, result := range out.MetricDataResults {
		if len(result.Values) == 0 {
			continue
		}

		value += result.Values[0]
	}

	return value, nil
}
