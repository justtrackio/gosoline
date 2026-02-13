package metric

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoCloudwatch "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/funk"
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
		Naming         CloudwatchNamingSettings `cfg:"naming"`
		Aggregate      bool                     `cfg:"aggregate" default:"true"`
		WriteGraceTime time.Duration            `cfg:"write_grace_time" default:"10s"`
	}

	CloudwatchNamingSettings struct {
		NamespacePattern   string `cfg:"namespace_pattern,nodecode" default:"{app.namespace}-{app.name}"`
		NamespaceDelimiter string `cfg:"namespace_delimiter" default:"/"`
	}

	cwWriterCtxKey string
	Dimensions     map[string]string
	StandardUnit   = types.StandardUnit

	cloudwatchWriter struct {
		logger         log.Logger
		clock          clock.Clock
		client         gosoCloudwatch.Client
		cwNamespace    string
		writeGraceTime time.Duration
	}
)

func ProvideCloudwatchWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	return appctx.Provide(ctx, cwWriterCtxKey("default"), func() (Writer, error) {
		return NewCloudwatchWriter(ctx, config, logger)
	})
}

func NewCloudwatchWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	testClock := clock.NewRealClock()

	var err error
	var cwSettings *CloudWatchSettings

	if cwSettings, err = getMetricWriterSettings[CloudWatchSettings](config, WriterTypeCloudwatch); err != nil {
		return nil, fmt.Errorf("failed to get cloudwatch settings: %w", err)
	}

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

	return NewCloudwatchWriterWithInterfaces(logger, testClock, client, cwNamespace, cwSettings.WriteGraceTime), nil
}

func NewCloudwatchWriterWithInterfaces(
	logger log.Logger,
	clock clock.Clock,
	cw gosoCloudwatch.Client,
	cwNamespace string,
	writeGraceTime time.Duration,
) Writer {
	return &cloudwatchWriter{
		logger:         logger.WithChannel("metrics"),
		clock:          clock,
		client:         cw,
		cwNamespace:    cwNamespace,
		writeGraceTime: writeGraceTime,
	}
}

func (w *cloudwatchWriter) GetPriority() int {
	return PriorityHigh
}

func (w *cloudwatchWriter) WriteOne(ctx context.Context, data *Datum) {
	w.Write(ctx, Data{data})
}

func (w *cloudwatchWriter) Write(applicationCtx context.Context, batch Data) {
	if len(batch) == 0 {
		return
	}

	delayedCtx, stop := exec.WithDelayedCancelContext(applicationCtx, w.writeGraceTime)
	defer stop()

	if err := w.write(delayedCtx, batch); err != nil {
		w.logger.Error(applicationCtx, "could not write to cloudwatch: %w", err)
	}
}

func (w *cloudwatchWriter) write(ctx context.Context, batch Data) error {
	metricData, err := w.buildMetricData(ctx, batch)

	logger := w.logger.WithFields(log.Fields{
		"namespace": w.cwNamespace,
	})

	if err != nil {
		logger.Info(ctx, "could not build metric data: %w", err)

		return nil
	}

	errs := &multierror.Error{}
	for _, chunk := range funk.Chunk(metricData, chunkSizeCloudWatch) {
		input := cloudwatch.PutMetricDataInput{
			MetricData: chunk,
			Namespace:  aws.String(w.cwNamespace),
		}

		if _, err = w.client.PutMetricData(ctx, &input); err != nil {
			errs = multierror.Append(errs, err)
		}
	}

	logger.Debug(ctx, "written %d metric data sets to cloudwatch", len(metricData))

	return errs.ErrorOrNil()
}

func (w *cloudwatchWriter) buildMetricData(ctx context.Context, batch Data) ([]types.MetricDatum, error) {
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
			w.logger.Error(ctx, "invalid metric dimension: %w", err)

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
	var err error
	var identity cfg.Identity
	var cloudwatchSettings *CloudWatchSettings
	var namespace string

	if identity, err = cfg.GetAppIdentity(config); err != nil {
		return "", fmt.Errorf("failed to get app identity from config: %w", err)
	}

	if cloudwatchSettings, err = getMetricWriterSettings[CloudWatchSettings](config, WriterTypeCloudwatch); err != nil {
		return "", fmt.Errorf("failed to get cloudwatch settings: %w", err)
	}

	if namespace, err = identity.Format(cloudwatchSettings.Naming.NamespacePattern, cloudwatchSettings.Naming.NamespaceDelimiter); err != nil {
		return "", fmt.Errorf("failed to format cloudwatch namespace: %w", err)
	}

	return namespace, nil
}
