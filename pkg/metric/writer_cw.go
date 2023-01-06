package metric

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoCloudwatch "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	PriorityLow  = 1
	PriorityHigh = 2

	UnitCount        = types.StandardUnitCount
	UnitSeconds      = types.StandardUnitSeconds
	UnitMilliseconds = types.StandardUnitMilliseconds

	chunkSizeCloudWatch = 20
	minusOneWeek        = -1 * 7 * 24 * time.Hour
	plusOneHour         = 1 * time.Hour
)

type (
	StandardUnit = types.StandardUnit
	Dimensions   map[string]string
)

type Datum struct {
	Priority   int          `json:"-"`
	Timestamp  time.Time    `json:"timestamp"`
	MetricName string       `json:"metricName"`
	Dimensions Dimensions   `json:"dimensions"`
	Value      float64      `json:"value"`
	Unit       StandardUnit `json:"unit"`
}

func (d *Datum) Id() string {
	return fmt.Sprintf("%s:%s", d.MetricName, d.DimensionKey())
}

func (d *Datum) DimensionKey() string {
	dims := make([]string, 0)

	for k, v := range d.Dimensions {
		flat := fmt.Sprintf("%s:%s", k, v)
		dims = append(dims, flat)
	}

	sort.Strings(dims)
	dimKey := strings.Join(dims, "-")

	return dimKey
}

func (d *Datum) IsValid() error {
	if d.MetricName == "" {
		return fmt.Errorf("missing metric name")
	}

	if d.Priority == 0 {
		return fmt.Errorf("metric %s has no priority", d.MetricName)
	}

	if d.Unit == "" {
		return fmt.Errorf("metric %s has no unit", d.MetricName)
	}

	return nil
}

type Data []*Datum

//go:generate mockery --name Writer
type Writer interface {
	GetPriority() int
	Write(batch Data)
	WriteOne(data *Datum)
}

type cwWriter struct {
	logger   log.Logger
	clock    clock.Clock
	client   gosoCloudwatch.Client
	settings *Settings
}

func NewCwWriter(ctx context.Context, config cfg.Config, logger log.Logger) (*cwWriter, error) {
	settings := getMetricSettings(config)
	testClock := clock.NewRealClock()

	client, err := gosoCloudwatch.ProvideClient(ctx, config, log.NewLogger(), "default", func(cfg *gosoCloudwatch.ClientConfig) {
		cfg.Settings.Backoff.MaxAttempts = 0
		cfg.Settings.Backoff.MaxElapsedTime = 60 * time.Second
		cfg.Settings.HttpClient.Timeout = 10 * time.Second
	})
	if err != nil {
		return nil, fmt.Errorf("can not create cloudwatch client: %w", err)
	}

	return NewCwWriterWithInterfaces(logger, testClock, client, settings), nil
}

func NewCwWriterWithInterfaces(logger log.Logger, clock clock.Clock, cw gosoCloudwatch.Client, settings *Settings) *cwWriter {
	return &cwWriter{
		logger:   logger.WithChannel("metrics"),
		clock:    clock,
		client:   cw,
		settings: settings,
	}
}

func (w *cwWriter) GetPriority() int {
	return PriorityHigh
}

func (w *cwWriter) WriteOne(data *Datum) {
	w.Write(Data{data})
}

func (w *cwWriter) Write(batch Data) {
	if !w.settings.Enabled || len(batch) == 0 {
		return
	}

	metricData, err := w.buildMetricData(batch)
	namespace := w.settings.Cloudwatch.Naming.Pattern
	values := map[string]string{
		"project": w.settings.Project,
		"env":     w.settings.Environment,
		"family":  w.settings.Family,
		"group":   w.settings.Group,
		"app":     w.settings.Application,
	}

	for key, val := range values {
		templ := fmt.Sprintf("{%s}", key)
		namespace = strings.ReplaceAll(namespace, templ, val)
	}

	logger := w.logger.WithFields(log.Fields{
		"namespace": namespace,
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
			Namespace:  aws.String(namespace),
		}

		if _, err = w.client.PutMetricData(context.Background(), &input); err != nil {
			logger.Info("could not write metric data: %s", err)
			continue
		}
	}

	logger.Debug("written %d metric data sets to cloudwatch", len(metricData))
}

func (w *cwWriter) buildMetricData(batch Data) ([]types.MetricDatum, error) {
	start := w.clock.Now().Add(minusOneWeek)
	end := w.clock.Now().Add(plusOneHour)
	metricData := make([]types.MetricDatum, 0, len(batch))

	for _, data := range batch {
		if data.Priority < w.GetPriority() {
			continue
		}

		if data.Timestamp.IsZero() {
			data.Timestamp = w.clock.Now()
		}

		if data.Timestamp.Before(start) || data.Timestamp.After(end) {
			continue
		}

		dimensions := make([]types.Dimension, 0)

		var err error
		for n, v := range data.Dimensions {
			if n == "" || v == "" {
				err = multierror.Append(err, fmt.Errorf("invalid dimension '%s' = '%s' for metric %s, this will later be rejected", n, v, data.MetricName))
			}

			dimensions = append(dimensions, types.Dimension{
				Name:  aws.String(n),
				Value: aws.String(v),
			})
		}

		if err != nil {
			w.logger.Error("invalid metric dimension: %w", err)
			continue
		}

		datum := types.MetricDatum{
			MetricName: aws.String(data.MetricName),
			Dimensions: dimensions,
			Timestamp:  aws.Time(data.Timestamp),
			Value:      aws.Float64(data.Value),
			Unit:       data.Unit,
		}

		metricData = append(metricData, datum)
	}

	return metricData, nil
}
