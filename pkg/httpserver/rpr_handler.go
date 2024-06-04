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
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/metric/calculator"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const PerRunnerMetricName = "HttpServerRequests"

func init() {
	calculator.RegisterHandlerFactory("httpserver_requests_per_runner", RequestsPerRunnerHandlerFactory)
}

type rprHandler struct {
	calculator.PerRunnerMetricHandler

	calculatorSettings *calculator.CalculatorSettings
	handlerSettings    *calculator.PerRunnerMetricSettings
	cwClient           gosoCloudwatch.Client
	clock              clock.Clock
	serverNames        []string
}

func RequestsPerRunnerHandlerFactory(ctx context.Context, config cfg.Config, logger log.Logger, calculatorSettings *calculator.CalculatorSettings) (calculator.Handler, error) {
	logger = logger.WithChannel("httpserver_requests_per_runner")
	settings := calculator.ReadHandlerSettings(config, "httpserver_requests_per_runner", &calculator.PerRunnerMetricSettings{})
	serverNames := getHttpServerNames(config)

	if !settings.Enabled {
		return nil, nil
	}

	if len(serverNames) == 0 {
		return nil, nil
	}

	var err error
	var cwClient gosoCloudwatch.Client

	if cwClient, err = gosoCloudwatch.ProvideClient(ctx, config, logger, "default"); err != nil {
		return nil, fmt.Errorf("can not create cloudwatch client: %w", err)
	}

	baseHandler := calculator.NewPerRunnerMetricHandlerWithInterfaces(logger, clock.Provider, cwClient, calculatorSettings)

	return NewRequestsPerRunnerHandlerWithInterfaces(clock.Provider, cwClient, baseHandler, calculatorSettings, settings, serverNames), nil
}

func NewRequestsPerRunnerHandlerWithInterfaces(
	clock clock.Clock,
	cwClient gosoCloudwatch.Client,
	baseHandler calculator.PerRunnerMetricHandler,
	calculatorSettings *calculator.CalculatorSettings,
	handlerSettings *calculator.PerRunnerMetricSettings,
	serverNames []string,
) calculator.Handler {
	return &rprHandler{
		PerRunnerMetricHandler: baseHandler,
		calculatorSettings:     calculatorSettings,
		handlerSettings:        handlerSettings,
		cwClient:               cwClient,
		clock:                  clock,
		serverNames:            serverNames,
	}
}

func (h *rprHandler) GetMetrics(ctx context.Context) (metric.Data, error) {
	var err error
	var requests float64
	var rpr *metric.Datum

	if requests, err = h.getRequestsMetrics(ctx); err != nil {
		return nil, fmt.Errorf("can not get number of requests received: %w", err)
	}

	if rpr, err = h.CalculatePerRunnerMetrics(ctx, PerRunnerMetricName, requests, h.handlerSettings); err != nil {
		return nil, fmt.Errorf("can not calculate httpserver per runner metrics: %w", err)
	}

	return metric.Data{rpr}, nil
}

func (h *rprHandler) getRequestsMetrics(ctx context.Context) (float64, error) {
	startTime := h.clock.Now().Add(-1 * h.handlerSettings.Period * 5)
	endTime := h.clock.Now().Add(-1 * h.handlerSettings.Period)
	period := int32(h.handlerSettings.Period.Seconds())

	queries := make([]types.MetricDataQuery, len(h.serverNames))
	for i, serverName := range h.serverNames {
		queries[i] = types.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m_%s", serverName)),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  aws.String(h.calculatorSettings.CloudWatchNamespace),
					MetricName: aws.String(MetricHttpRequestCount),
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("ServerName"),
							Value: aws.String(serverName),
						},
					},
				},
				Period: aws.Int32(period),
				Stat:   aws.String(string(types.StatisticSum)),
				Unit:   types.StandardUnitCount,
			},
		}
	}

	input := &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		MetricDataQueries: queries,
	}

	var err error
	var out *cloudwatch.GetMetricDataOutput
	var requests float64

	if out, err = h.cwClient.GetMetricData(ctx, input); err != nil {
		return 0, fmt.Errorf("can not get metric: %w", err)
	}

	if len(out.MetricDataResults) == 0 {
		return 0, fmt.Errorf("no metric results")
	}

	for _, result := range out.MetricDataResults {
		if len(result.Values) == 0 {
			return 0, fmt.Errorf("no metric values for %s", *result.Id)
		}

		requests += result.Values[0]
	}

	return requests, nil
}

func getHttpServerNames(config cfg.Config) []string {
	names := maps.Keys(config.GetStringMap("httpserver"))
	names = slices.DeleteFunc(names, func(s string) bool {
		return s == "health-check"
	})

	return names
}
