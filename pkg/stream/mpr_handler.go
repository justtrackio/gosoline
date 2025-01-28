package stream

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
)

const (
	PrmHandlerName              = "stream_messages_per_runner"
	PerRunnerMetricName         = "StreamMessages"
	MessagesAvailableMetricName = "StreamMessagesAvailable"
	MessagesSentMetricName      = "StreamMessagesSent"
)

func init() {
	calculator.RegisterHandlerFactory(PrmHandlerName, MessagesPerRunnerHandlerFactory)
}

type mprHandler struct {
	calculator.PerRunnerMetricHandler
	logger log.Logger

	calculatorSettings *calculator.CalculatorSettings
	handlerSettings    *calculator.PerRunnerMetricSettings
	cwClient           gosoCloudwatch.Client
	clock              clock.Clock
	queueNames         []string
}

func MessagesPerRunnerHandlerFactory(ctx context.Context, config cfg.Config, logger log.Logger, calculatorSettings *calculator.CalculatorSettings) (calculator.Handler, error) {
	logger = logger.WithChannel(PrmHandlerName)
	settings := calculator.ReadHandlerSettings(config, PrmHandlerName, &calculator.PerRunnerMetricSettings{})

	if !settings.Enabled {
		return nil, nil
	}

	var err error
	var cwClient gosoCloudwatch.Client
	var queueNames []string

	if cwClient, err = gosoCloudwatch.ProvideClient(ctx, config, logger, "default"); err != nil {
		return nil, fmt.Errorf("can not create cloudwatch client: %w", err)
	}

	if queueNames, err = getQueueNames(config); err != nil {
		return nil, fmt.Errorf("can not get queue names: %w", err)
	}

	if len(queueNames) == 0 {
		return nil, nil
	}

	baseHandler := calculator.NewPerRunnerMetricHandlerWithInterfaces(logger, clock.Provider, cwClient, calculatorSettings)

	return NewMessagesPerRunnerHandlerWithInterfaces(logger, clock.Provider, cwClient, baseHandler, calculatorSettings, settings, queueNames), nil
}

func NewMessagesPerRunnerHandlerWithInterfaces(
	logger log.Logger,
	clock clock.Clock,
	cwClient gosoCloudwatch.Client,
	baseHandler calculator.PerRunnerMetricHandler,
	calculatorSettings *calculator.CalculatorSettings,
	handlerSettings *calculator.PerRunnerMetricSettings,
	queueNames []string,
) calculator.Handler {
	return &mprHandler{
		logger:                 logger,
		PerRunnerMetricHandler: baseHandler,
		calculatorSettings:     calculatorSettings,
		handlerSettings:        handlerSettings,
		cwClient:               cwClient,
		clock:                  clock,
		queueNames:             queueNames,
	}
}

func (h *mprHandler) createDatum(metricName string, value float64) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityLow,
		MetricName: metricName,
		Value:      value,
		Unit:       metric.UnitCount,
	}
}

func (h *mprHandler) GetMetrics(ctx context.Context) (metric.Data, error) {
	var err error
	var messagesTotal, messagesSent, messagesAvailable float64
	var rpr *metric.Datum

	if messagesSent, messagesAvailable, err = h.getMessagesMetric(ctx); err != nil {
		return nil, fmt.Errorf("can not get number of messages: %w", err)
	}

	messagesTotal = messagesSent + messagesAvailable

	ma := h.createDatum(MessagesAvailableMetricName, messagesAvailable)
	ms := h.createDatum(MessagesSentMetricName, messagesSent)

	if rpr, err = h.CalculatePerRunnerMetrics(ctx, PerRunnerMetricName, messagesTotal, h.handlerSettings); err != nil {
		h.logger.Warn("can not calculate metrics per runner for handler: can not calculate httpserver per runner metrics: %s: %T", err.Error(), *h)

		return metric.Data{
			ma,
			ms,
		}, nil
	}

	return metric.Data{
		rpr,
		ma,
		ms,
	}, nil
}

func (h *mprHandler) getMessagesMetric(ctx context.Context) (messagesSent float64, messagesVisible float64, err error) {
	if messagesSent, err = h.getQueueMetrics(ctx, "NumberOfMessagesSent", types.StatisticSum); err != nil {
		return 0, 0, fmt.Errorf("can not get number of messages sent: %w", err)
	}

	if messagesVisible, err = h.getQueueMetrics(ctx, "ApproximateNumberOfMessagesVisible", types.StatisticMaximum); err != nil {
		return 0, 0, fmt.Errorf("can not get number of messages visible: %w", err)
	}

	return messagesSent, messagesVisible, nil
}

func (h *mprHandler) getQueueMetrics(ctx context.Context, metric string, stat types.Statistic) (float64, error) {
	startTime := h.clock.Now().Add(-1 * h.handlerSettings.Period * 5)
	endTime := h.clock.Now().Add(-1 * h.handlerSettings.Period)
	period := int32(h.handlerSettings.Period.Seconds())
	queries := make([]types.MetricDataQuery, len(h.queueNames))

	for i, queueName := range h.queueNames {
		queries[i] = types.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m_%d", i)),
			MetricStat: &types.MetricStat{
				Metric: &types.Metric{
					Namespace:  aws.String("AWS/SQS"),
					MetricName: aws.String(metric),
					Dimensions: []types.Dimension{
						{
							Name:  aws.String("QueueName"),
							Value: aws.String(queueName),
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
