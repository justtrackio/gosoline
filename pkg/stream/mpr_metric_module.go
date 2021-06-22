package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/conc"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/metric"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"time"
)

const (
	metricNameStreamMprMessagesPerRunner = "StreamMprMessagesPerRunner"
)

type MessagesPerRunnerMetricWriterSettings struct {
	QueueNames         []string
	UpdatePeriod       time.Duration
	TargetValue        float64
	MaxIncreasePercent float64
	MaxIncreasePeriod  time.Duration
	AppId              cfg.AppId
	MemberId           string
}

func MessagesPerRunnerMetricWriterFactory(config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
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
	leaderElection conc.LeaderElection
	cwClient       cloudwatchiface.CloudWatchAPI
	metricWriter   metric.Writer
	clock          clock.Clock
	ticker         clock.Ticker
	settings       *MessagesPerRunnerMetricWriterSettings
}

func NewMessagesPerRunnerMetricWriter(settings *MessagesPerRunnerMetricSettings) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var err error
		var queueNames []string
		var leaderElection conc.LeaderElection

		logger = logger.WithChannel("stream-metric-messages-per-runner")

		if queueNames, err = getQueueNames(config); err != nil {
			return nil, fmt.Errorf("can't create stream-metric-messages-per-runner: %w", err)
		}

		writerSettings := &MessagesPerRunnerMetricWriterSettings{
			QueueNames:         queueNames,
			UpdatePeriod:       settings.Period,
			TargetValue:        settings.TargetValue,
			MaxIncreasePercent: settings.MaxIncreasePercent,
			MaxIncreasePeriod:  settings.MaxIncreasePeriod,
			AppId:              cfg.AppId{},
			MemberId:           uuid.New().NewV4(),
		}
		writerSettings.AppId.PadFromConfig(config)

		if leaderElection, err = conc.NewLeaderElection(config, logger, settings.LeaderElection); err != nil {
			return nil, fmt.Errorf("can not create leader election for stream-metric-messages-per-runner writer: %w", err)
		}

		cwClient := metric.ProvideCloudWatchClient(config)
		metricWriter := metric.NewDaemonWriter()
		ticker := clock.NewRealTicker(settings.Period)

		return NewMessagesPerRunnerMetricWriterWithInterfaces(logger, leaderElection, cwClient, metricWriter, clock.Provider, ticker, writerSettings)
	}
}

func NewMessagesPerRunnerMetricWriterWithInterfaces(logger log.Logger, leaderElection conc.LeaderElection, cwClient cloudwatchiface.CloudWatchAPI, metricWriter metric.Writer, clock clock.Clock, ticker clock.Ticker, settings *MessagesPerRunnerMetricWriterSettings) (*MessagesPerRunnerMetricWriter, error) {
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
		case <-u.ticker.Tick():
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

	if messagesPerRunner, err = u.calculateMessagesPerRunner(); err != nil {
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

func (u *MessagesPerRunnerMetricWriter) calculateMessagesPerRunner() (float64, error) {
	var err error
	var runnerCount, messagesSent, messagesVisible, currentMpr, newMpr, maxMpr float64

	if messagesSent, err = u.getQueueMetrics("NumberOfMessagesSent", cloudwatch.StatisticSum); err != nil {
		return 0, fmt.Errorf("can not get number of messages sent: %w", err)
	}

	if messagesVisible, err = u.getQueueMetrics("ApproximateNumberOfMessagesVisible", cloudwatch.StatisticMaximum); err != nil {
		return 0, fmt.Errorf("can not get number of messages visible: %w", err)
	}

	if runnerCount, err = u.getEcsMetric("DesiredTaskCount", cloudwatch.StatisticMaximum, u.settings.UpdatePeriod); err != nil {
		return 0, fmt.Errorf("can not get runner count: %w", err)
	}

	if runnerCount == 0 {
		return 0, fmt.Errorf("runner count is zero")
	}

	if currentMpr, err = u.getStreamMprMetric(metricNameStreamMprMessagesPerRunner, cloudwatch.StatisticAverage, u.settings.MaxIncreasePeriod); err != nil {
		u.logger.Warn("can not get current messages per runner metric: defaulting to 0")
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

func (u *MessagesPerRunnerMetricWriter) getQueueMetrics(metric string, stat string) (float64, error) {
	startTime := u.clock.Now().Add(-1 * u.settings.UpdatePeriod * 5)
	endTime := u.clock.Now().Add(-1 * u.settings.UpdatePeriod)
	period := int64(u.settings.UpdatePeriod.Seconds())
	queries := make([]*cloudwatch.MetricDataQuery, len(u.settings.QueueNames))

	for i, queueName := range u.settings.QueueNames {
		queries[i] = &cloudwatch.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m%d", i)),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String("AWS/SQS"),
					MetricName: aws.String(metric),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("QueueName"),
							Value: aws.String(queueName),
						},
					},
				},
				Period: aws.Int64(period),
				Stat:   aws.String(stat),
				Unit:   aws.String(cloudwatch.StandardUnitCount),
			},
		}
	}

	input := &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(startTime),
		EndTime:           aws.Time(endTime),
		MetricDataQueries: queries,
	}

	out, err := u.cwClient.GetMetricData(input)

	if err != nil {
		return 0, fmt.Errorf("can not get metric data: %w", err)
	}

	value := 0.0
	for _, result := range out.MetricDataResults {
		if len(result.Values) == 0 {
			continue
		}

		value += *result.Values[0]
	}

	return value, nil
}

func (u *MessagesPerRunnerMetricWriter) getStreamMprMetric(name string, stat string, period time.Duration) (float64, error) {
	appId := u.settings.AppId
	namespace := fmt.Sprintf("%s/%s/%s/%s", appId.Project, appId.Environment, appId.Family, appId.Application)

	startTime := u.clock.Now().Add(-1 * period)
	endTime := u.clock.Now().Add(-1 * u.settings.UpdatePeriod)
	periodSeconds := int64(u.settings.UpdatePeriod.Seconds())

	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String(namespace),
						MetricName: aws.String(name),
					},
					Period: aws.Int64(periodSeconds),
					Stat:   aws.String(stat),
					Unit:   aws.String(cloudwatch.StandardUnitCount),
				},
			},
		},
		MaxDatapoints: aws.Int64(1),
	}

	out, err := u.cwClient.GetMetricData(input)

	if err != nil {
		return 0, fmt.Errorf("can not get metric: %w", err)
	}

	if len(out.MetricDataResults) == 0 {
		return 0, fmt.Errorf("no metric results")
	}

	if len(out.MetricDataResults[0].Values) == 0 {
		return 0, fmt.Errorf("no metric values")
	}

	value := *out.MetricDataResults[0].Values[0]

	return value, nil
}

func (u *MessagesPerRunnerMetricWriter) getEcsMetric(name string, stat string, period time.Duration) (float64, error) {
	appId := u.settings.AppId
	clusterName := fmt.Sprintf("%s-%s-%s", appId.Project, appId.Environment, appId.Family)

	startTime := u.clock.Now().Add(-1 * period * 5)
	endTime := u.clock.Now().Add(-1 * period)
	periodSeconds := int64(period.Seconds())

	input := &cloudwatch.GetMetricDataInput{
		StartTime: aws.Time(startTime),
		EndTime:   aws.Time(endTime),
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String("ECS/ContainerInsights"),
						MetricName: aws.String(name),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("ClusterName"),
								Value: aws.String(clusterName),
							},
							{
								Name:  aws.String("ServiceName"),
								Value: aws.String(appId.Application),
							},
						},
					},
					Period: aws.Int64(periodSeconds),
					Stat:   aws.String(stat),
					Unit:   aws.String(cloudwatch.StandardUnitCount),
				},
			},
		},
		MaxDatapoints: aws.Int64(1),
	}

	out, err := u.cwClient.GetMetricData(input)

	if err != nil {
		return 0, fmt.Errorf("can not get metric: %w", err)
	}

	if len(out.MetricDataResults) == 0 {
		return 0, fmt.Errorf("no metric results")
	}

	if len(out.MetricDataResults[0].Values) == 0 {
		return 0, fmt.Errorf("no metric values")
	}

	value := *out.MetricDataResults[0].Values[0]

	return value, nil
}
