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
	"sort"
	"time"
)

func TrafficMetricWriterFactory(config cfg.Config, logger mon.Logger) (map[string]kernel.ModuleFactory, error) {
	modules := map[string]kernel.ModuleFactory{}
	consumerSettings := readAllConsumerSettings(config)

	for consumerName := range consumerSettings {
		settings := consumerSettings[consumerName]

		if !settings.TrafficMetric.Enabled {
			continue
		}

		moduleName := fmt.Sprintf("consumer-%s-traffic-metric", consumerName)
		modules[moduleName] = NewTrafficMetricWriter(consumerName, &settings.TrafficMetric)
	}

	return modules, nil
}

type TrafficMetricWriterSettings struct {
	Enabled      bool          `cfg:"enabled" default:"false"`
	Period       time.Duration `cfg:"period" default:"1m"`
	AppId        cfg.AppId
	ConsumerName string
	QueueName    string
	MemberId     string
	RunnerCount  int
}

type TrafficMetricWriter struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger         mon.Logger
	leaderElection conc.LeaderElection
	cwClient       cloudwatchiface.CloudWatchAPI
	metricWriter   mon.MetricWriter
	clock          clock.Clock
	settings       *TrafficMetricWriterSettings
}

func NewTrafficMetricWriter(consumerName string, settings *TrafficMetricWriterSettings) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
		channelName := fmt.Sprintf("consumer-%s-traffic-metric", consumerName)
		logger = logger.WithChannel(channelName)

		appId := &cfg.AppId{}
		appId.PadFromConfig(config)

		consumerSettings := readConsumerSettings(config, consumerName)
		inputType := readInputType(config, consumerSettings.Input)

		if inputType != InputTypeSqs {
			return nil, fmt.Errorf("can not create traffic metric writer as consumer input is not of type SQS")
		}

		inputSettings := readSqsInputSettings(config, consumerSettings.Input)

		settings.AppId = *appId
		settings.ConsumerName = consumerName
		settings.QueueName = sqs.QueueName(inputSettings)
		settings.MemberId = uuid.New().NewV4()
		settings.RunnerCount = consumerSettings.RunnerCount

		tableName := fmt.Sprintf("%s-%s-%s-traffic-metric-writer-leaders", appId.Project, appId.Environment, appId.Family)
		groupId := fmt.Sprintf("%s-%s", appId.Application, consumerName)

		leaderElection, err := conc.NewDdbLeaderElection(config, logger, &conc.DdbLeaderElectionSettings{
			TableName:     tableName,
			GroupId:       groupId,
			LeaseDuration: settings.Period,
		})

		if err != nil {
			return nil, fmt.Errorf("can not create leader election for traffic metric writer of consumer %s: %w", consumerName, err)
		}

		cwClient := mon.ProvideCloudWatchClient(config)
		metricWriter := mon.NewMetricDaemonWriter()

		return NewTrafficMetricWriterWithInterfaces(logger, leaderElection, cwClient, metricWriter, clock.Provider, settings)
	}
}

func NewTrafficMetricWriterWithInterfaces(logger mon.Logger, leaderElection conc.LeaderElection, cwClient cloudwatchiface.CloudWatchAPI, metricWriter mon.MetricWriter, clock clock.Clock, settings *TrafficMetricWriterSettings) (*TrafficMetricWriter, error) {
	writer := &TrafficMetricWriter{
		logger:         logger,
		leaderElection: leaderElection,
		cwClient:       cwClient,
		metricWriter:   metricWriter,
		clock:          clock,
		settings:       settings,
	}

	return writer, nil
}

func (u *TrafficMetricWriter) Run(ctx context.Context) error {
	u.writeRunnerCountMetric(ctx)
	u.writeTrafficMetric(ctx)

	ticker := clock.NewRealTicker(u.settings.Period)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.Tick():
			u.writeRunnerCountMetric(ctx)
			u.writeTrafficMetric(ctx)
		}
	}
}

func (u *TrafficMetricWriter) writeRunnerCountMetric(ctx context.Context) {
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

func (u *TrafficMetricWriter) writeTrafficMetric(ctx context.Context) {
	var err error
	var isLeader bool
	var traffic float64

	if isLeader, err = u.leaderElection.IsLeader(ctx, u.settings.MemberId); err != nil {
		u.logger.Warnf("will assume leader role as election failed: %s", err)
		isLeader = true
	}

	if !isLeader {
		u.logger.Infof("not leading: do nothing")
		return
	}

	if traffic, err = u.calculateTraffic(); err != nil {
		u.logger.Warnf("can not calculate traffic: %s", err)
		return
	}

	u.metricWriter.WriteOne(&mon.MetricDatum{
		Priority:   mon.PriorityHigh,
		Timestamp:  u.clock.Now(),
		MetricName: metricNameConsumerTraffic,
		Dimensions: map[string]string{
			"Consumer": u.settings.ConsumerName,
		},
		Unit:  mon.UnitCountAverage,
		Value: traffic,
	})
}

func (u *TrafficMetricWriter) calculateTraffic() (float64, error) {
	var err error
	var runnerCount, messagesSent, messagesVisible, traffic float64

	if messagesSent, err = u.getMessagesSent(); err != nil {
		return 0, fmt.Errorf("can not get number of messages sent: %w", err)
	}

	if messagesVisible, err = u.getMessagesVisible(); err != nil {
		return 0, fmt.Errorf("can not get number of messages visible: %w", err)
	}

	if runnerCount, err = u.getRunnerCount(); err != nil {
		return 0, fmt.Errorf("can not get runner count: %w", err)
	}

	if runnerCount == 0 {
		return 0, fmt.Errorf("runner count is zero")
	}

	traffic = (messagesSent + messagesVisible) / runnerCount

	u.logger.WithFields(mon.Fields{
		"messagesSent":    messagesSent,
		"messagesVisible": messagesVisible,
		"runnerCount":     runnerCount,
		"traffic":         traffic,
	}).Infof("traffic is at %f", traffic)

	return traffic, nil
}

func (u *TrafficMetricWriter) getMessagesSent() (float64, error) {
	startTime := u.clock.Now().Add(-1 * u.settings.Period * 5)
	endTime := u.clock.Now().Add(-1 * u.settings.Period)
	period := int64(u.settings.Period.Seconds())

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/SQS"),
		MetricName: aws.String("NumberOfMessagesSent"),
		Dimensions: []*cloudwatch.Dimension{
			{
				Name:  aws.String("QueueName"),
				Value: aws.String(u.settings.QueueName),
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
		return 0, fmt.Errorf("can not get metric statistics: %w", err)
	}

	if len(out.Datapoints) == 0 {
		return 0, fmt.Errorf("no data points available")
	}

	sort.Slice(out.Datapoints, func(i, j int) bool {
		return out.Datapoints[i].Timestamp.After(*out.Datapoints[j].Timestamp)
	})
	messagesSent := *out.Datapoints[0].Sum

	return messagesSent, nil
}

func (u *TrafficMetricWriter) getMessagesVisible() (float64, error) {
	startTime := u.clock.Now().Add(-1 * u.settings.Period * 5)
	endTime := u.clock.Now().Add(-1 * u.settings.Period)
	period := int64(u.settings.Period.Seconds())

	input := &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String("AWS/SQS"),
		MetricName: aws.String("ApproximateNumberOfMessagesVisible"),
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

	sort.Slice(out.Datapoints, func(i, j int) bool {
		return out.Datapoints[i].Timestamp.After(*out.Datapoints[j].Timestamp)
	})
	messagesVisible := *out.Datapoints[0].Maximum

	return messagesVisible, nil
}

func (u *TrafficMetricWriter) getRunnerCount() (float64, error) {
	appId := u.settings.AppId
	namespace := fmt.Sprintf("%s/%s/%s/%s", appId.Project, appId.Environment, appId.Family, appId.Application)

	startTime := u.clock.Now().Add(-1 * u.settings.Period * 5)
	endTime := u.clock.Now().Add(-1 * u.settings.Period)
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

	sort.Slice(out.Datapoints, func(i, j int) bool {
		return out.Datapoints[i].Timestamp.After(*out.Datapoints[j].Timestamp)
	})
	runnerCount := *out.Datapoints[0].Sum

	if runnerCount == 0 {
		return 0, fmt.Errorf("invalid runner count of 0")
	}

	return runnerCount, nil
}
