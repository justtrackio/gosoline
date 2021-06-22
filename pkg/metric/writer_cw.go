package metric

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/hashicorp/go-multierror"
	"github.com/jonboulle/clockwork"
	"sort"
	"strings"
	"time"
)

const (
	PriorityLow  = 1
	PriorityHigh = 2

	UnitCount               = cloudwatch.StandardUnitCount
	UnitCountAverage        = "UnitCountAverage"
	UnitSeconds             = cloudwatch.StandardUnitSeconds
	UnitSecondsAverage      = "UnitSecondsAverage"
	UnitMilliseconds        = cloudwatch.StandardUnitMilliseconds
	UnitMillisecondsAverage = "UnitMillisecondsAverage"

	chunkSizeCloudWatch = 20
	minusOneWeek        = -1 * 7 * 24 * time.Hour
	plusOneHour         = 1 * time.Hour
)

type Dimensions map[string]string

type Datum struct {
	Priority   int        `json:"-"`
	Timestamp  time.Time  `json:"timestamp"`
	MetricName string     `json:"metricName"`
	Dimensions Dimensions `json:"dimensions"`
	Value      float64    `json:"value"`
	Unit       string     `json:"unit"`
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
	clock    clockwork.Clock
	cw       cloudwatchiface.CloudWatchAPI
	settings *Settings
}

func NewCwWriter(config cfg.Config, logger log.Logger) (*cwWriter, error) {
	settings := getMetricSettings(config)

	clock := clockwork.NewRealClock()
	cw := ProvideCloudWatchClient(config)

	return NewCwWriterWithInterfaces(logger, clock, cw, settings), nil
}

func NewCwWriterWithInterfaces(logger log.Logger, clock clockwork.Clock, cw cloudwatchiface.CloudWatchAPI, settings *Settings) *cwWriter {
	return &cwWriter{
		logger:   logger.WithChannel("metrics"),
		clock:    clock,
		cw:       cw,
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
	namespace := fmt.Sprintf("%s/%s/%s/%s", w.settings.Project, w.settings.Environment, w.settings.Family, w.settings.Application)

	if err != nil {
		w.logger.WithFields(log.Fields{
			"namespace": namespace,
		}).Error("could not write metric data: %w", err)

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

		_, err := w.cw.PutMetricData(&input)

		if err != nil {
			w.logger.Error("could not write metric data: %w", err)
			continue
		}
	}

	w.logger.Debug("written %d metric data sets to cloudwatch", len(metricData))
}

func (w *cwWriter) buildMetricData(batch Data) ([]*cloudwatch.MetricDatum, error) {
	start := w.clock.Now().Add(minusOneWeek)
	end := w.clock.Now().Add(plusOneHour)
	metricData := make([]*cloudwatch.MetricDatum, 0, len(batch))

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

		dimensions := make([]*cloudwatch.Dimension, 0)

		var err error
		for n, v := range data.Dimensions {
			if n == "" || v == "" {
				err = multierror.Append(err, fmt.Errorf("invalid dimension '%s' = '%s' for metric %s, this will later be rejected", n, v, data.MetricName))
			}

			dimensions = append(dimensions, &cloudwatch.Dimension{
				Name:  aws.String(n),
				Value: aws.String(v),
			})
		}

		if err != nil {
			w.logger.Error("invalid metric dimension: %w", err)
			continue
		}

		datum := &cloudwatch.MetricDatum{
			MetricName: aws.String(data.MetricName),
			Dimensions: dimensions,
			Timestamp:  aws.Time(data.Timestamp),
			Value:      aws.Float64(data.Value),

			Unit: aws.String(data.Unit),
		}

		if err := datum.Validate(); err != nil {
			w.logger.Error("invalid metric datum: %w", err)
			continue
		}

		metricData = append(metricData, datum)
	}

	return metricData, nil
}
