package metric

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoCloudwatch "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/log"
)

func init() {
	RegisterWriterFactory(WriterTypeCloudwatch, ProvideCloudwatchWriter)
}

var _ Writer = &cloudwatchWriter{}

const (
	UnitCount        = types.StandardUnitCount
	UnitSeconds      = types.StandardUnitSeconds
	UnitMilliseconds = types.StandardUnitMilliseconds

	chunkSizeCloudWatch = 20
	minusOneWeek        = -1 * 7 * 24 * time.Hour
	plusOneHour         = 1 * time.Hour
)

type (
	CloudWatchSettings struct {
		Naming    CloudwatchNamingSettings `cfg:"naming"`
		Aggregate bool                     `cfg:"aggregate" default:"true"`
	}

	CloudwatchNamingSettings struct {
		Pattern string `cfg:"pattern,nodecode" default:"{project}/{env}/{family}/{group}-{app}"`
	}

	cwWriterCtxKey string
	Dimensions     map[string]string
	StandardUnit   = types.StandardUnit

	cloudwatchWriter struct {
		logger      log.Logger
		clock       clock.Clock
		client      gosoCloudwatch.Client
		cwNamespace string
	}
)

func ProvideCloudwatchWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	return appctx.Provide(ctx, cwWriterCtxKey("default"), func() (Writer, error) {
		return NewCloudwatchWriter(ctx, config, logger)
	})
}

func NewCloudwatchWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	testClock := clock.NewRealClock()

	cwNamespace, err := GetCloudWatchNamespace(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get cloudwatch namespace: %w", err)
	}

	client, err := gosoCloudwatch.ProvideClient(ctx, config, log.NewLogger(), "default", func(cfg *gosoCloudwatch.ClientConfig) {
		cfg.Settings.Backoff.MaxAttempts = 0
		cfg.Settings.Backoff.MaxElapsedTime = 60 * time.Second
		cfg.Settings.HttpClient.Timeout = 10 * time.Second
	})
	if err != nil {
		return nil, fmt.Errorf("can not create cloudwatch client: %w", err)
	}

	return NewCloudwatchWriterWithInterfaces(logger, testClock, client, cwNamespace), nil
}

func NewCloudwatchWriterWithInterfaces(logger log.Logger, clock clock.Clock, cw gosoCloudwatch.Client, cwNamespace string) Writer {
	return &cloudwatchWriter{
		logger:      logger.WithChannel("metrics"),
		clock:       clock,
		client:      cw,
		cwNamespace: cwNamespace,
	}
}

func (w *cloudwatchWriter) GetPriority() int {
	return PriorityHigh
}

func (w *cloudwatchWriter) WriteOne(data *Datum) {
	w.Write(Data{data})
}

func (w *cloudwatchWriter) Write(batch Data) {
	if len(batch) == 0 {
		return
	}

	metricData, err := w.buildMetricData(batch)

	logger := w.logger.WithFields(log.Fields{
		"namespace": w.cwNamespace,
	})

	if err != nil {
		logger.Info("could not build metric data: %w", err)

		return
	}

	for i := 0; i < len(metricData); i += chunkSizeCloudWatch {
		end := i + chunkSizeCloudWatch

		if end > len(metricData) {
			end = len(metricData)
		}

		input := cloudwatch.PutMetricDataInput{
			MetricData: metricData[i:end],
			Namespace:  aws.String(w.cwNamespace),
		}

		if _, err = w.client.PutMetricData(context.Background(), &input); err != nil {
			logger.Info("could not write metric data: %s", err)

			continue
		}
	}

	logger.Debug("written %d metric data sets to cloudwatch", len(metricData))
}

func (w *cloudwatchWriter) buildMetricData(batch Data) ([]types.MetricDatum, error) {
	start := w.clock.Now().Add(minusOneWeek)
	end := w.clock.Now().Add(plusOneHour)
	metricData := make([]types.MetricDatum, 0, len(batch))

	for _, data := range batch {
		if data.Priority < w.GetPriority() {
			continue
		}

		timestamp := aws.Time(w.clock.Now())
		if !data.Timestamp.IsZero() {
			timestamp = aws.Time(data.Timestamp)
		}

		if timestamp.Before(start) || timestamp.After(end) {
			continue
		}

		dimensions := make([]types.Dimension, 0)

		var err error
		for name, value := range data.Dimensions {
			if value == DimensionDefault {
				continue
			}

			if name == "" || value == "" {
				err = multierror.Append(err, fmt.Errorf("invalid dimension '%s' = '%s' for metric %s, this will later be rejected", name, value, data.MetricName))
			}

			dimensions = append(dimensions, types.Dimension{
				Name:  aws.String(name),
				Value: aws.String(value),
			})
		}

		if err != nil {
			w.logger.Error("invalid metric dimension: %w", err)

			continue
		}

		datum := types.MetricDatum{
			MetricName: aws.String(data.MetricName),
			Dimensions: dimensions,
			Timestamp:  timestamp,
			Value:      aws.Float64(data.Value),
			Unit:       data.Unit,
		}

		metricData = append(metricData, datum)
	}

	return metricData, nil
}

func GetCloudWatchNamespace(config cfg.Config) (string, error) {
	appId, err := cfg.GetAppIdFromConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to get app id from config: %w", err)
	}

	values := map[string]string{
		"project": appId.Project,
		"env":     appId.Environment,
		"family":  appId.Family,
		"group":   appId.Group,
		"app":     appId.Application,
	}

	cloudwatchSettings := &CloudWatchSettings{}
	if err := getMetricWriterSettings(config, WriterTypeCloudwatch, cloudwatchSettings); err != nil {
		return "", fmt.Errorf("failed to get cloudwatch settings: %w", err)
	}
	namespace := cloudwatchSettings.Naming.Pattern

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		namespace = strings.ReplaceAll(namespace, templ, val)
	}

	return namespace, nil
}
