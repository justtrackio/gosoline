package stream

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoCloudwatch "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	metricNameStreamMprMessagesPerRunner = "StreamMprMessagesPerRunner"
)

type MessagesPerRunnerMetricWriterSettings struct {
	Ecs                 MessagesPerRunnerEcsSettings
	MaxIncreasePeriod   time.Duration
	UpdatePeriod        time.Duration
	CloudwatchNamespace string
	MaxIncreasePercent  float64
	MemberId            string
	QueueNames          []string
	TargetValue         float64
}

func MessagesPerRunnerMetricWriterFactory(_ context.Context, config cfg.Config, _ log.Logger) (map[string]kernel.ModuleFactory, error) {
	settings := readMessagesPerRunnerMetricSettings(config)
	modules := map[string]kernel.ModuleFactory{}

	if !settings.Enabled {
		return modules, nil
	}

	moduleName := "stream-metric-messages-per-runner"
	modules[moduleName] = NewMessagesPerRunnerMetricWriter(settings)

	return modules, nil
}

type MessagesPerRunnerMetricWriter struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger         log.Logger
	leaderElection ddb.LeaderElection
	cwClient       gosoCloudwatch.Client
	metricWriter   metric.Writer
	clock          clock.Clock
	ticker         clock.Ticker
	settings       *MessagesPerRunnerMetricWriterSettings
}

func NewMessagesPerRunnerMetricWriter(settings *MessagesPerRunnerMetricSettings) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var err error
		var queueNames []string
		var leaderElection ddb.LeaderElection
		var cwClient gosoCloudwatch.Client

		logger = logger.WithChannel("stream-metric-messages-per-runner")

		if queueNames, err = getQueueNames(config); err != nil {
			return nil, fmt.Errorf("can't create stream-metric-messages-per-runner: %w", err)
		}

		if len(queueNames) == 0 {
			return nil, fmt.Errorf("failed to detect any SQS queues to monitor")
		}

		cwNamespace := getCloudwatchNamespace(config, settings.Cloudwatch.Naming.Pattern)

		writerSettings := &MessagesPerRunnerMetricWriterSettings{
			CloudwatchNamespace: cwNamespace,
			QueueNames:          queueNames,
			UpdatePeriod:        settings.Period,
			TargetValue:         settings.TargetValue,
			MaxIncreasePercent:  settings.MaxIncreasePercent,
			MaxIncreasePeriod:   settings.MaxIncreasePeriod,
			Ecs:                 settings.Ecs,
			MemberId:            uuid.New().NewV4(),
		}

		if leaderElection, err = ddb.NewLeaderElection(ctx, config, logger, settings.LeaderElection); err != nil {
			return nil, fmt.Errorf("can not create leader election for stream-metric-messages-per-runner writer: %w", err)
		}

		if cwClient, err = gosoCloudwatch.ProvideClient(ctx, config, logger, "default"); err != nil {
			return nil, fmt.Errorf("can not create cloudwatch client: %w", err)
		}

		metricWriter := metric.NewWriter()
		ticker := clock.NewRealTicker(settings.Period)

		return NewMessagesPerRunnerMetricWriterWithInterfaces(logger, leaderElection, cwClient, metricWriter, clock.Provider, ticker, writerSettings)
	}
}

func getCloudwatchNamespace(config cfg.Config, cwNamespacePattern string) string {
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		cwNamespacePattern = strings.ReplaceAll(cwNamespacePattern, templ, val)
	}

	return cwNamespacePattern
}

func NewMessagesPerRunnerMetricWriterWithInterfaces(logger log.Logger, leaderElection ddb.LeaderElection, cwClient gosoCloudwatch.Client, metricWriter metric.Writer, clock clock.Clock, ticker clock.Ticker, settings *MessagesPerRunnerMetricWriterSettings) (*MessagesPerRunnerMetricWriter, error) {
	writer := &MessagesPerRunnerMetricWriter{
		logger:         logger,
		leaderElection: leaderElection,
		cwClient:       cwClient,
		metricWriter:   metricWriter,
		clock:          clock,
		ticker:         ticker,
		settings:       settings,
	}

	return writer, nil
}

func (u *MessagesPerRunnerMetricWriter) Run(ctx context.Context) error {
	if err := u.writeMessagesPerRunnerMetric(ctx); err != nil {
		return fmt.Errorf("can not write message per runner metric: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-u.ticker.Chan():
			if err := u.writeMessagesPerRunnerMetric(ctx); err != nil {
				return fmt.Errorf("can not write message per runner metric: %w", err)
			}
		}
	}
}

func (u *MessagesPerRunnerMetricWriter) writeMessagesPerRunnerMetric(ctx context.Context) error {
	var err error
	var isLeader bool
	var messagesPerRunner float64

	if isLeader, err = u.leaderElection.IsLeader(ctx, u.settings.MemberId); err != nil {
		if conc.IsLeaderElectionFatalError(err) {
			return fmt.Errorf("can not decide on leader: %w", err)
		}

		u.logger.Warn("will assume leader role as election failed: %s", err)
		isLeader = true
	}

	if !isLeader {
		u.logger.Info("not leading: do nothing")
		return nil
	}

	if messagesPerRunner, err = u.calculateMessagesPerRunner(ctx); err != nil {
		u.logger.Warn("can not calculate messages per runner: %s", err)
		return nil
	}

	u.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		Timestamp:  u.clock.Now(),
		MetricName: metricNameStreamMprMessagesPerRunner,
		Unit:       metric.UnitCountAverage,
		Value:      messagesPerRunner,
	})

	return nil
}

func (u *MessagesPerRunnerMetricWriter) calculateMessagesPerRunner(ctx context.Context) (float64, error) {
	var err error
	var runnerCount, messagesSent, messagesVisible, currentMpr, newMpr, maxMpr float64

	if messagesSent, err = u.getQueueMetrics(ctx, "NumberOfMessagesSent", types.StatisticSum); err != nil {
		return 0, fmt.Errorf("can not get number of messages sent: %w", err)
	}

	if messagesVisible, err = u.getQueueMetrics(ctx, "ApproximateNumberOfMessagesVisible", types.StatisticMaximum); err != nil {
		return 0, fmt.Errorf("can not get number of messages visible: %w", err)
	}

	if runnerCount, err = u.getEcsMetric(ctx, "DesiredTaskCount", types.StatisticMaximum, u.settings.UpdatePeriod); err != nil {
		return 0, fmt.Errorf("can not get runner count: %w", err)
	}

	if runnerCount == 0 {
		return 0, fmt.Errorf("runner count is zero")
	}

	if currentMpr, err = u.getStreamMprMetric(ctx, metricNameStreamMprMessagesPerRunner, types.StatisticAverage, u.settings.MaxIncreasePeriod); err != nil {
		u.logger.Warn("can not get current messages per runner metric: %s, defaulting to 0", err.Error())
		currentMpr = 0
	}

	newMpr = (messagesSent + messagesVisible) / runnerCount

	if currentMpr == 0 {
		currentMpr = newMpr
	}

	maxMpr = currentMpr * (u.settings.MaxIncreasePercent / 100)

	if currentMpr < u.settings.TargetValue {
		maxMpr = u.settings.TargetValue * (u.settings.MaxIncreasePercent / 100)
	}

	if newMpr > maxMpr {
		u.logger.Warn("newMpr of %f is higher than configured maxMpr of %f: falling back to max", newMpr, maxMpr)
		newMpr = maxMpr
	}

	u.logger.WithFields(log.Fields{
		"messagesSent":      messagesSent,
		"messagesVisible":   messagesVisible,
		"runnerCount":       runnerCount,
		"messagesPerRunner": newMpr,
	}).Info("%f messages per runner", newMpr)

	return newMpr, nil
}

func (u *MessagesPerRunnerMetricWriter) getQueueMetrics(ctx context.Context, metric string, stat types.Statistic) (float64, error) {
	startTime := u.clock.Now().Add(-1 * u.settings.UpdatePeriod * 5)
	endTime := u.clock.Now().Add(-1 * u.settings.UpdatePeriod)
	period := int32(u.settings.UpdatePeriod.Seconds())
	queries := make([]types.MetricDataQuery, len(u.settings.QueueNames))

	for i, queueName := range u.settings.QueueNames {
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

	out, err := u.cwClient.GetMetricData(ctx, input)
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

func (u *MessagesPerRunnerMetricWriter) getStreamMprMetric(ctx context.Context, name string, stat types.Statistic, period time.Duration) (float64, error) {
	namespace := u.settings.CloudwatchNamespace

	startTime := u.clock.Now().Add(-1 * period)
	endTime := u.clock.Now().Add(-1 * u.settings.UpdatePeriod)
	periodSeconds := int32(u.settings.UpdatePeriod.Seconds())

	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		MetricDataQueries: []types.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &types.MetricStat{
					Metric: &types.Metric{
						Namespace:  aws.String(namespace),
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

	out, err := u.cwClient.GetMetricData(ctx, input)
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

func (u *MessagesPerRunnerMetricWriter) getEcsMetric(ctx context.Context, name string, stat types.Statistic, period time.Duration) (float64, error) {
	clusterName := u.settings.Ecs.Cluster
	serviceName := u.settings.Ecs.Service

	startTime := u.clock.Now().Add(-1 * period * 5)
	endTime := u.clock.Now().Add(-1 * period)
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

	out, err := u.cwClient.GetMetricData(ctx, input)
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
