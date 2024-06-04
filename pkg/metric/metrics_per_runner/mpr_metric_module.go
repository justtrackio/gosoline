package metrics_per_runner

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
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	metricNameMprMetricsPerRunner = "%sMprMetricsPerRunner"
)

type MetricsPerRunnerMetricWriterSettings struct {
	Ecs                 MetricsPerRunnerEcsSettings
	CloudwatchNamespace string
	MemberId            string
	Handlers            map[string]InitializedHandler
}

type InitializedHandler struct {
	Handler
	Settings HandlerSettings
}

func MetricsPerRunnerMetricWriterFactory(_ context.Context, config cfg.Config, _ log.Logger) (map[string]kernel.ModuleFactory, error) {
	enabledHandlers := map[string]Handler{}

	for name, handler := range handlers {
		if !handler.IsEnabled(config) {
			continue
		}

		enabledHandlers[name] = handler
	}

	if len(enabledHandlers) == 0 {
		return map[string]kernel.ModuleFactory{}, nil
	}

	settings := readMetricsPerRunnerMetricSettings(config)
	modules := map[string]kernel.ModuleFactory{}

	moduleName := "metrics-per-runner"
	modules[moduleName] = NewMetricsPerRunnerMetricWriter(settings, enabledHandlers)

	return modules, nil
}

type MetricsPerRunnerMetricWriter struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger         log.Logger
	leaderElection ddb.LeaderElection
	cwClient       gosoCloudwatch.Client
	metricWriter   metric.Writer
	clock          clock.Clock
	ticker         clock.Ticker
	settings       *MetricsPerRunnerMetricWriterSettings
}

func NewMetricsPerRunnerMetricWriter(settings *MetricsPerRunnerMetricSettings, enabledHandlers map[string]Handler) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		var err error
		var leaderElection ddb.LeaderElection
		var cwClient gosoCloudwatch.Client

		logger = logger.WithChannel("metrics-per-runner")

		cwNamespace := getCloudwatchNamespace(config, settings.Cloudwatch.Naming.Pattern)

		initializedHandlers := map[string]InitializedHandler{}
		var period *time.Duration
		var periodHandlerName string
		for name, handler := range enabledHandlers {
			var handlerSettings *HandlerSettings

			if handlerSettings, err = handler.Init(ctx, config, logger, cwNamespace); err != nil {
				return nil, fmt.Errorf("can't initialize %s metrics-per-runner handler: %w", name, err)
			}

			initializedHandlers[name] = InitializedHandler{
				Handler:  handler,
				Settings: *handlerSettings,
			}

			if period == nil {
				period = mdl.Box(handlerSettings.Period)
				periodHandlerName = name
			} else if *period != handlerSettings.Period {
				return nil, fmt.Errorf("handler %s has different metrics per runner period from handler %s", name, periodHandlerName)
			}
		}

		writerSettings := &MetricsPerRunnerMetricWriterSettings{
			CloudwatchNamespace: cwNamespace,
			Ecs:                 settings.Ecs,
			Handlers:            initializedHandlers,
			MemberId:            uuid.New().NewV4(),
		}

		if leaderElection, err = ddb.NewLeaderElection(ctx, config, logger, settings.LeaderElection); err != nil {
			return nil, fmt.Errorf("can not create leader election for metrics-per-runner writer: %w", err)
		}

		if cwClient, err = gosoCloudwatch.ProvideClient(ctx, config, logger, "default"); err != nil {
			return nil, fmt.Errorf("can not create cloudwatch client: %w", err)
		}

		metricWriter := metric.NewWriter()
		ticker := clock.NewRealTicker(*period)

		return NewMetricsPerRunnerMetricWriterWithInterfaces(logger, leaderElection, cwClient, metricWriter, clock.Provider, ticker, writerSettings), nil
	}
}

func NewMetricsPerRunnerMetricWriterWithInterfaces(logger log.Logger, leaderElection ddb.LeaderElection, cwClient gosoCloudwatch.Client, metricWriter metric.Writer, clock clock.Clock, ticker clock.Ticker, settings *MetricsPerRunnerMetricWriterSettings) *MetricsPerRunnerMetricWriter {
	return &MetricsPerRunnerMetricWriter{
		logger:         logger,
		leaderElection: leaderElection,
		cwClient:       cwClient,
		metricWriter:   metricWriter,
		clock:          clock,
		ticker:         ticker,
		settings:       settings,
	}
}

func (u *MetricsPerRunnerMetricWriter) Run(ctx context.Context) error {

	for {
		if err := u.writeMetricsPerRunnerMetric(ctx); err != nil {
			return fmt.Errorf("can not write message per runner metric: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-u.ticker.Chan():
			continue
		}
	}
}

func (u *MetricsPerRunnerMetricWriter) writeMetricsPerRunnerMetric(ctx context.Context) error {
	var err error
	var isLeader bool

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

	for name, handler := range u.settings.Handlers {
		var metricsPerRunner float64

		if metricsPerRunner, err = u.calculateMetricsPerRunner(ctx, name, handler); err != nil {
			u.logger.Warn("can not calculate metrics per runner: %s", err)

			return nil
		}

		u.metricWriter.WriteOne(&metric.Datum{
			Priority:   metric.PriorityHigh,
			Timestamp:  u.clock.Now(),
			MetricName: fmt.Sprintf(metricNameMprMetricsPerRunner, name),
			Unit:       metric.UnitCountAverage,
			Value:      metricsPerRunner,
		})
	}

	return nil
}

func (u *MetricsPerRunnerMetricWriter) calculateMetricsPerRunner(ctx context.Context, name string, handler InitializedHandler) (float64, error) {
	var err error
	var runnerCount, metricSum, currentMpr, newMpr, maxMpr float64

	if metricSum, err = handler.GetMetricSum(ctx); err != nil {
		return 0, fmt.Errorf("can not get %s metric: %w", name, err)
	}

	if runnerCount, err = u.getEcsMetric(ctx, "DesiredTaskCount", types.StatisticMaximum, handler.Settings.Period); err != nil {
		return 0, fmt.Errorf("can not get runner count: %w", err)
	}

	if runnerCount == 0 {
		return 0, fmt.Errorf("runner count is zero")
	}

	if currentMpr, err = u.getPreviousMetric(ctx, fmt.Sprintf(metricNameMprMetricsPerRunner, name), handler.Settings, types.StatisticAverage); err != nil {
		u.logger.Warn("can not get current %s metric per runner metric: %s, defaulting to 0", name, err.Error())
		currentMpr = 0
	}

	newMpr = metricSum / runnerCount

	if currentMpr == 0 {
		currentMpr = newMpr
	}

	maxMpr = currentMpr * (handler.Settings.MaxIncreasePercent / 100)

	if currentMpr < handler.Settings.TargetValue {
		maxMpr = handler.Settings.TargetValue * (handler.Settings.MaxIncreasePercent / 100)
	}

	if newMpr > maxMpr {
		u.logger.Warn("newMpr of %f is higher than configured maxMpr of %f: falling back to max", newMpr, maxMpr)
		newMpr = maxMpr
	}

	u.logger.WithFields(log.Fields{
		"handler":           name,
		"metricSum":         metricSum,
		"runnerCount":       runnerCount,
		"messagesPerRunner": newMpr,
	}).Info("%f %s metrics per runner", newMpr, name)

	return newMpr, nil
}

func (u *MetricsPerRunnerMetricWriter) getPreviousMetric(ctx context.Context, name string, settings HandlerSettings, stat types.Statistic) (float64, error) {
	namespace := u.settings.CloudwatchNamespace

	startTime := u.clock.Now().Add(-1 * settings.MaxIncreasePeriod)
	endTime := u.clock.Now().Add(-1 * settings.Period)
	periodSeconds := int32(settings.Period.Seconds())

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

func (u *MetricsPerRunnerMetricWriter) getEcsMetric(ctx context.Context, name string, stat types.Statistic, period time.Duration) (float64, error) {
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
