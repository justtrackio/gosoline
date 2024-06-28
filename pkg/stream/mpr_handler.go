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
	"github.com/justtrackio/gosoline/pkg/metric/metrics_per_runner"
)

func init() {
	metrics_per_runner.RegisterHandler("stream", &mprHandler{})
}

type mprHandler struct {
	settings   MessagesPerRunnerMetricSettings
	queueNames []string
	cwClient   gosoCloudwatch.Client
	clock      clock.Clock
}

func (h *mprHandler) IsEnabled(config cfg.Config) bool {
	return messagesPerRunnerIsEnabled(config)
}

func (h *mprHandler) Init(ctx context.Context, config cfg.Config, logger log.Logger, cwNamespace string) (*metrics_per_runner.HandlerSettings, error) {
	var err error

	h.settings = readMessagesPerRunnerMetricSettings(config)
	h.clock = clock.Provider

	if h.queueNames, err = getQueueNames(config); err != nil {
		return nil, fmt.Errorf("can't create stream-metric-messages-per-runner: %w", err)
	}

	if len(h.queueNames) == 0 {
		return nil, fmt.Errorf("failed to detect any SQS queues to monitor")
	}

	if h.cwClient, err = gosoCloudwatch.ProvideClient(ctx, config, logger, "default"); err != nil {
		return nil, fmt.Errorf("can not create cloudwatch client: %w", err)
	}

	return &h.settings.HandlerSettings, nil
}

func (h *mprHandler) GetMetricSum(ctx context.Context) (float64, error) {
	var err error
	var messagesSent, messagesVisible float64

	if messagesSent, err = h.getQueueMetrics(ctx, "NumberOfMessagesSent", types.StatisticSum); err != nil {
		return 0, fmt.Errorf("can not get number of messages sent: %w", err)
	}

	if messagesVisible, err = h.getQueueMetrics(ctx, "ApproximateNumberOfMessagesVisible", types.StatisticMaximum); err != nil {
		return 0, fmt.Errorf("can not get number of messages visible: %w", err)
	}

	return messagesSent + messagesVisible, nil
}

func (h *mprHandler) getQueueMetrics(ctx context.Context, metric string, stat types.Statistic) (float64, error) {
	startTime := h.clock.Now().Add(-1 * h.settings.Period * 5)
	endTime := h.clock.Now().Add(-1 * h.settings.Period)
	period := int32(h.settings.Period.Seconds())
	queries := make([]types.MetricDataQuery, len(h.queueNames))

	for i, queueName := range h.queueNames {
		queries[i] = types.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m%d", i)),
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
