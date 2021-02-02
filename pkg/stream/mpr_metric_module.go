package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/conc"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/uuid"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"sort"
	"time"
)

const (
	metricNameStreamMprRunnerCount    = "StreamMprRunnerCount"
	metricNameStreamMessagesPerRunner = "StreamMprMessagesPerRunner"
)

type MessagesPerRunnerMetricWriterSettings struct {
	Name          string
	ConsumerSpecs []*ConsumerSpec
	Period        time.Duration
	AppId         cfg.AppId
	MemberId      string
}

func MessagesPerRunnerMetricWriterFactory(config cfg.Config, logger mon.Logger) (map[string]kernel.ModuleFactory, error) {
	modules := map[string]kernel.ModuleFactory{}
	mprSettings := readAllMessagesPerRunnerMetricSettings(config)

	for mprName := range mprSettings {
		settings := mprSettings[mprName]

		moduleName := fmt.Sprintf("stream-metric-messages-per-runner-%s", mprName)
		modules[moduleName] = NewMessagesPerRunnerMetricWriter(mprName, settings)
	}

	return modules, nil
}

type MessagesPerRunnerMetricWriter struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger         mon.Logger
	leaderElection conc.LeaderElection
	cwClient       cloudwatchiface.CloudWatchAPI
	metricWriter   mon.MetricWriter
	clock          clock.Clock
	settings       *MessagesPerRunnerMetricWriterSettings
}

func NewMessagesPerRunnerMetricWriter(mprName string, settings *MessagesPerRunnerMetricSettings) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
		var err error
		var consumerSpecs []*ConsumerSpec

		channelName := fmt.Sprintf("stream-metric-messages-per-runner-%s", mprName)
		logger = logger.WithChannel(channelName)

		if consumerSpecs, err = GetConsumerSpecs(config, settings.Consumers); err != nil {
			return nil, fmt.Errorf("can't create stream-metric-messages-per-runner-%s: %w", mprName, err)
		}

		writerSettings := &MessagesPerRunnerMetricWriterSettings{
			Name:          mprName,
			ConsumerSpecs: consumerSpecs,
			Period:        settings.Period,
			AppId:         cfg.AppId{},
			MemberId:      uuid.New().NewV4(),
		}
		writerSettings.AppId.PadFromConfig(config)

		tableName := fmt.Sprintf("%s-%s-%s-stream-metric-writer-leaders", writerSettings.AppId.Project, writerSettings.AppId.Environment, writerSettings.AppId.Family)
		groupId := fmt.Sprintf("%s-%s", writerSettings.AppId.Application, mprName)

		leaderElection, err := conc.NewDdbLeaderElection(config, logger, &conc.DdbLeaderElectionSettings{
			TableName:     tableName,
			GroupId:       groupId,
			LeaseDuration: settings.Period,
		})

		if err != nil {
			return nil, fmt.Errorf("can not create leader election for stream-metric-messages-per-runner writer %s: %w", mprName, err)
		}

		cwClient := mon.ProvideCloudWatchClient(config)
		metricWriter := mon.NewMetricDaemonWriter()

		return NewMessagesPerRunnerMetricWriterWithInterfaces(logger, leaderElection, cwClient, metricWriter, clock.Provider, writerSettings)
	}
}

func NewMessagesPerRunnerMetricWriterWithInterfaces(logger mon.Logger, leaderElection conc.LeaderElection, cwClient cloudwatchiface.CloudWatchAPI, metricWriter mon.MetricWriter, clock clock.Clock, settings *MessagesPerRunnerMetricWriterSettings) (*MessagesPerRunnerMetricWriter, error) {
	writer := &MessagesPerRunnerMetricWriter{
		logger:         logger,
		leaderElection: leaderElection,
		cwClient:       cwClient,
		metricWriter:   metricWriter,
		clock:          clock,
		settings:       settings,
	}

	return writer, nil
}

func (u *MessagesPerRunnerMetricWriter) Run(ctx context.Context) error {
	u.writeRunnerCountMetric(ctx)
	u.writeMessagesPerRunnerMetric(ctx)

	ticker := clock.NewRealTicker(u.settings.Period)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.Tick():
			u.writeRunnerCountMetric(ctx)
			u.writeMessagesPerRunnerMetric(ctx)
		}
	}
}

func (u *MessagesPerRunnerMetricWriter) writeRunnerCountMetric(ctx context.Context) {
	runnerCount := 0

	for _, spec := range u.settings.ConsumerSpecs {
		runnerCount += spec.RunnerCount
	}

	u.metricWriter.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  u.clock.Now(),
		MetricName: metricNameStreamMprRunnerCount,
		Dimensions: map[string]string{
			"Name": u.settings.Name,
		},
		Unit:  mon.UnitCount,
		Value: float64(runnerCount),
	})
}

func (u *MessagesPerRunnerMetricWriter) writeMessagesPerRunnerMetric(ctx context.Context) {
	var err error
	var isLeader bool
	var messagesPerRunner float64

	if isLeader, err = u.leaderElection.IsLeader(ctx, u.settings.MemberId); err != nil {
		u.logger.Warnf("will assume leader role as election failed: %s", err)
		isLeader = true
	}

	if !isLeader {
		u.logger.Infof("not leading: do nothing")
		//return
	}

	if messagesPerRunner, err = u.calculateMessagesPerRunner(); err != nil {
		u.logger.Warnf("can not calculate messages per runner: %s", err)
		return
	}

	u.metricWriter.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  u.clock.Now(),
		MetricName: metricNameStreamMessagesPerRunner,
		Dimensions: map[string]string{
			"Name": u.settings.Name,
		},
		Unit:  mon.UnitCountAverage,
		Value: messagesPerRunner,
	})
}

func (u *MessagesPerRunnerMetricWriter) calculateMessagesPerRunner() (float64, error) {
	var err error
	var runnerCount, messagesSent, messagesVisible, messagesPerRunner float64

	if messagesSent, err = u.getQueueMetrics("NumberOfMessagesSent", cloudwatch.StatisticSum); err != nil {
		return 0, fmt.Errorf("can not get number of messages sent: %w", err)
	}

	if messagesVisible, err = u.getQueueMetrics("ApproximateNumberOfMessagesVisible", cloudwatch.StatisticMaximum); err != nil {
		return 0, fmt.Errorf("can not get number of messages visible: %w", err)
	}

	if runnerCount, err = u.getRunnerCount(); err != nil {
		return 0, fmt.Errorf("can not get runner count: %w", err)
	}

	if runnerCount == 0 {
		return 0, fmt.Errorf("runner count is zero")
	}

	messagesPerRunner = (messagesSent + messagesVisible) / runnerCount

	u.logger.WithFields(mon.Fields{
		"messagesSent":      messagesSent,
		"messagesVisible":   messagesVisible,
		"runnerCount":       runnerCount,
		"messagesPerRunner": messagesPerRunner,
	}).Infof("%f messages per runner", messagesPerRunner)

	return messagesPerRunner, nil
}

func (u *MessagesPerRunnerMetricWriter) getQueueMetrics(metric string, stat string) (float64, error) {
	startTime := u.clock.Now().Add(-1 * u.settings.Period * 5)
	endTime := u.clock.Now().Add(-1 * u.settings.Period)
	period := int64(u.settings.Period.Seconds())
	queries := make([]*cloudwatch.MetricDataQuery, len(u.settings.ConsumerSpecs))

	for i, spec := range u.settings.ConsumerSpecs {
		queries[i] = &cloudwatch.MetricDataQuery{
			Id: aws.String(fmt.Sprintf("m%d", i)),
			MetricStat: &cloudwatch.MetricStat{
				Metric: &cloudwatch.Metric{
					Namespace:  aws.String("AWS/SQS"),
					MetricName: aws.String(metric),
					Dimensions: []*cloudwatch.Dimension{
						{
							Name:  aws.String("QueueName"),
							Value: aws.String(spec.QueueName),
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
		value += *result.Values[0]
	}

	return value, nil
}

func (u *MessagesPerRunnerMetricWriter) getRunnerCount() (float64, error) {
	appId := u.settings.AppId
	namespace := fmt.Sprintf("%s/%s/%s/%s", appId.Project, appId.Environment, appId.Family, appId.Application)

	startTime := u.clock.Now().Add(-1 * u.settings.Period * 5)
	endTime := u.clock.Now().Add(-1 * u.settings.Period)
	period := int64(u.settings.Period.Seconds())

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metricNameStreamMprRunnerCount),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("Name"),
				Value: aws.String(u.settings.Name),
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

	sort.Slice(out.Datapoints, func(i, j int) bool {
		return out.Datapoints[i].Timestamp.After(*out.Datapoints[j].Timestamp)
	})
	runnerCount := *out.Datapoints[0].Sum

	if runnerCount == 0 {
		return 0, fmt.Errorf("invalid runner count of 0")
	}

	return runnerCount, nil
}
