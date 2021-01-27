package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/conc"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sqs"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"time"
)

func BacklogMetricWriterFactory(config cfg.Config, logger mon.Logger) (map[string]kernel.ModuleFactory, error) {
	modules := map[string]kernel.ModuleFactory{}
	consumerSettings := readAllConsumerSettings(config)

	for consumerName := range consumerSettings {
		settings := consumerSettings[consumerName]

		if !settings.BacklogMetric.Enabled {
			continue
		}

		moduleName := fmt.Sprintf("consumer-%s-backlog-metric", consumerName)
		modules[moduleName] = NewBacklogMetricWriter(consumerName, &settings.BacklogMetric)
	}

	return modules, nil
}

type BacklogMetricWriterSettings struct {
	Enabled      bool          `cfg:"enabled" default:"false"`
	Period       time.Duration `cfg:"period" default:"1m"`
	AppId        cfg.AppId
	ConsumerName string
	QueueName    string
	MemberId     string
	RunnerCount  int
}

type BacklogMetricWriter struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger         mon.Logger
	leaderElection conc.LeaderElection
	cwClient       cloudwatchiface.CloudWatchAPI
	metricWriter   mon.MetricWriter
	clock          clock.Clock
	settings       *BacklogMetricWriterSettings
}

func NewBacklogMetricWriter(consumerName string, settings *BacklogMetricWriterSettings) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
		channelName := fmt.Sprintf("consumer-%s-backlog-metric", consumerName)
		logger = logger.WithChannel(channelName)

		appId := &cfg.AppId{}
		appId.PadFromConfig(config)

		consumerSettings := readConsumerSettings(config, consumerName)
		inputType := readInputType(config, consumerSettings.Input)

		if inputType != InputTypeSqs {
			return nil, fmt.Errorf("can not create backlog metric writer as consumer input is not of type SQS")
		}

		inputSettings := readSqsInputSettings(config, consumerSettings.Input)

		settings.AppId = *appId
		settings.ConsumerName = consumerName
		settings.QueueName = sqs.QueueName(inputSettings)
		settings.MemberId = uuid.New().NewV4()
		settings.RunnerCount = consumerSettings.RunnerCount

		tableName := fmt.Sprintf("%s-%s-%s-backlog-metric-writer-leaders", appId.Project, appId.Environment, appId.Family)
		groupId := fmt.Sprintf("%s-%s", appId.Application, consumerName)

		leaderElection, err := conc.NewDdbLeaderElection(config, logger, &conc.DdbLeaderElectionSettings{
			TableName:     tableName,
			GroupId:       groupId,
			LeaseDuration: settings.Period,
		})

		if err != nil {
			return nil, fmt.Errorf("can not create leader election for backlog metric writer of consumer %s: %w", consumerName, err)
		}

		cwClient := mon.ProvideCloudWatchClient(config)
		metricWriter := mon.NewMetricDaemonWriter()

		return NewBacklogMetricWriterWithInterfaces(logger, leaderElection, cwClient, metricWriter, clock.Provider, settings)
	}
}

func NewBacklogMetricWriterWithInterfaces(logger mon.Logger, leaderElection conc.LeaderElection, cwClient cloudwatchiface.CloudWatchAPI, metricWriter mon.MetricWriter, clock clock.Clock, settings *BacklogMetricWriterSettings) (*BacklogMetricWriter, error) {
	writer := &BacklogMetricWriter{
		logger:         logger,
		leaderElection: leaderElection,
		cwClient:       cwClient,
		metricWriter:   metricWriter,
		clock:          clock,
		settings:       settings,
	}

	return writer, nil
}

func (u *BacklogMetricWriter) Run(ctx context.Context) error {
	u.writeRunnerCountMetric(ctx)
	u.writeBacklogMetric(ctx)

	ticker := clock.NewRealTicker(u.settings.Period)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.Tick():
			u.writeRunnerCountMetric(ctx)
			u.writeBacklogMetric(ctx)
		}
	}
}

func (u *BacklogMetricWriter) writeRunnerCountMetric(ctx context.Context) {
	u.metricWriter.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  u.clock.Now(),
		MetricName: metricNameConsumerRunnerCount,
		Dimensions: map[string]string{
			"Consumer": u.settings.ConsumerName,
		},
		Unit:  mon.UnitCount,
		Value: float64(u.settings.RunnerCount),
	})
}

func (u *BacklogMetricWriter) writeBacklogMetric(ctx context.Context) {
	var err error
	var isLeader bool
	var backlog float64

	if isLeader, err = u.leaderElection.IsLeader(ctx, u.settings.MemberId); err != nil {
		u.logger.Warnf("will assume leader role as election failed: %s", err)
		isLeader = true
	}

	if !isLeader {
		u.logger.Infof("not leading: do nothing")
		return
	}

	if backlog, err = u.calculateBacklog(); err != nil {
		u.logger.Warnf("can not calculate backlog: %s", err)
		return
	}

	u.metricWriter.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  u.clock.Now(),
		MetricName: metricNameConsumerBacklog,
		Dimensions: map[string]string{
			"Consumer": u.settings.ConsumerName,
		},
		Unit:  mon.UnitSeconds,
		Value: backlog,
	})
}

func (u *BacklogMetricWriter) calculateBacklog() (float64, error) {
	var err error
	var runnerCount, consumeDuration, messagesNotVisible, messagesVisible, messagesInQueue, backlog float64

	if messagesNotVisible, err = u.getMessagesInQueue("ApproximateNumberOfMessagesNotVisible"); err != nil {
		return 0, fmt.Errorf("can not get number of messages not visible: %w", err)
	}

	if messagesVisible, err = u.getMessagesInQueue("ApproximateNumberOfMessagesVisible"); err != nil {
		return 0, fmt.Errorf("can not get number of messages visible: %w", err)
	}

	messagesInQueue = messagesNotVisible + messagesVisible

	if messagesInQueue == 0 {
		return 0, nil
	}

	if runnerCount, err = u.getRunnerCount(); err != nil {
		return 0, fmt.Errorf("can not get runner count: %w", err)
	}

	if consumeDuration, err = u.getConsumeDuration(); err != nil {
		return 0, fmt.Errorf("can not get consume duration: %w", err)
	}

	backlog = messagesInQueue / runnerCount * consumeDuration

	return backlog, nil
}

func (u *BacklogMetricWriter) getMessagesInQueue(metricName string) (float64, error) {
	startTime := u.clock.Now().Add(-1 * u.settings.Period * 5)
	endTime := u.clock.Now()
	period := int64(u.settings.Period.Seconds())

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/SQS"),
		MetricName: aws.String(metricName),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("QueueName"),
				Value: aws.String(u.settings.QueueName),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int64(period),
		Statistics: []*string{aws.String(cloudwatch.StatisticMaximum)},
		Unit:       aws.String(cloudwatch.StandardUnitCount),
	}

	out, err := u.cwClient.GetMetricStatistics(input)

	if err != nil {
		return 0, fmt.Errorf("can not get metric statistics: %w", err)
	}

	if len(out.Datapoints) == 0 {
		return 0, fmt.Errorf("no data points available")
	}

	consumeDuration := *out.Datapoints[len(out.Datapoints)-1].Maximum

	return consumeDuration, nil
}

func (u *BacklogMetricWriter) getRunnerCount() (float64, error) {
	appId := u.settings.AppId
	namespace := fmt.Sprintf("%s/%s/%s/%s", appId.Project, appId.Environment, appId.Family, appId.Application)

	startTime := u.clock.Now().Add(-1 * u.settings.Period * 5)
	endTime := u.clock.Now()
	period := int64(u.settings.Period.Seconds())

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricNameConsumerRunnerCount),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("Consumer"),
				Value: aws.String(u.settings.ConsumerName),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int64(period),
		Statistics: []*string{aws.String(cloudwatch.StatisticSum)},
		Unit:       aws.String(cloudwatch.StandardUnitCount),
	}

	out, err := u.cwClient.GetMetricStatistics(input)

	if err != nil {
		return 0, fmt.Errorf("can not get metric: %w", err)
	}

	if len(out.Datapoints) == 0 {
		return 0, fmt.Errorf("no data points available")
	}

	runnerCount := *out.Datapoints[len(out.Datapoints)-1].Sum

	if runnerCount == 0 {
		return 0, fmt.Errorf("invalid runner count of 0")
	}

	return runnerCount, nil
}

func (u *BacklogMetricWriter) getConsumeDuration() (float64, error) {
	appId := u.settings.AppId
	namespace := fmt.Sprintf("%s/%s/%s/%s", appId.Project, appId.Environment, appId.Family, appId.Application)

	startTime := u.clock.Now().Add(-1 * u.settings.Period * 5)
	endTime := u.clock.Now()
	period := int64(u.settings.Period.Seconds())

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricNameConsumerDuration),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("Consumer"),
				Value: aws.String(u.settings.ConsumerName),
			},
		},
		StartTime:  aws.Time(startTime),
		EndTime:    aws.Time(endTime),
		Period:     aws.Int64(period),
		Statistics: []*string{aws.String(cloudwatch.StatisticAverage)},
		Unit:       aws.String(cloudwatch.StandardUnitMilliseconds),
	}

	out, err := u.cwClient.GetMetricStatistics(input)

	if err != nil {
		return 0, fmt.Errorf("can not get metric for consumer runner count: %w", err)
	}

	if len(out.Datapoints) == 0 {
		return 0, fmt.Errorf("no data points available")
	}

	consumeDuration := *out.Datapoints[len(out.Datapoints)-1].Average / 1000

	if consumeDuration == 0 {
		return 0, fmt.Errorf("invalid consume duration of 0")
	}

	return consumeDuration, nil
}
