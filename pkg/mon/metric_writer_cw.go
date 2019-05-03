package mon

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/jonboulle/clockwork"
	"time"
)

const (
	PriorityLow  = 0
	PriorityHigh = 1

	UnitCount        = cloudwatch.StandardUnitCount
	UnitCountAverage = "UnitCountAverage"
	UnitSeconds      = cloudwatch.StandardUnitSeconds
	UnitMilliseconds = cloudwatch.StandardUnitMilliseconds

	chunkSizeCloudWatch = 20
	minusOneWeek        = -1 * 7 * 24 * time.Hour
	plusOneHour         = 1 * time.Hour
)

type MetricDimensions map[string]string

type MetricDatum struct {
	Priority   int              `json:"-"`
	Timestamp  time.Time        `json:"timestamp"`
	MetricName string           `json:"metricName"`
	Dimensions MetricDimensions `json:"dimensions"`
	Value      float64          `json:"value"`
	Unit       string           `json:"unit"`
}
type MetricData []*MetricDatum

type MetricSettings struct {
	cfg.AppId
	Enabled bool
}

//go:generate mockery -name MetricWriter
type MetricWriter interface {
	GetPriority() int
	Write(batch MetricData)
	WriteOne(data *MetricDatum)
}

type cwWriter struct {
	logger   Logger
	clock    clockwork.Clock
	cw       cloudwatchiface.CloudWatchAPI
	settings MetricSettings
}

func NewMetricCwWriter(config cfg.Config, logger Logger) *cwWriter {
	settings := MetricSettings{}
	settings.PadFromConfig(config)
	settings.Enabled = config.GetBool("metric_enabled")

	clock := clockwork.NewRealClock()
	cw := ProvideCloudWatchClient(config)

	return NewMetricCwWriterWithInterfaces(logger, clock, cw, settings)
}

func NewMetricCwWriterWithInterfaces(logger Logger, clock clockwork.Clock, cw cloudwatchiface.CloudWatchAPI, settings MetricSettings) *cwWriter {
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

func (w *cwWriter) WriteOne(data *MetricDatum) {
	w.Write(MetricData{data})
}

func (w *cwWriter) Write(batch MetricData) {
	if !w.settings.Enabled || len(batch) == 0 {
		return
	}

	metricData, err := w.buildMetricData(batch)
	namespace := fmt.Sprintf("%s/%s/%s/%s", w.settings.Project, w.settings.Environment, w.settings.Family, w.settings.Application)

	if err != nil {
		w.logger.WithFields(Fields{
			"namespace": namespace,
		}).Error(err, "could not write metric data")

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
			w.logger.Error(err, "could not write metric data")
			continue
		}
	}

	w.logger.Debugf("written %d metric data sets to cloudwatch", len(metricData))
}

func (w *cwWriter) buildMetricData(batch MetricData) ([]*cloudwatch.MetricDatum, error) {
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

		for n, v := range data.Dimensions {
			dimensions = append(dimensions, &cloudwatch.Dimension{
				Name:  aws.String(n),
				Value: aws.String(v),
			})
		}

		datum := &cloudwatch.MetricDatum{
			MetricName: aws.String(data.MetricName),
			Dimensions: dimensions,
			Timestamp:  aws.Time(data.Timestamp),
			Value:      aws.Float64(data.Value),

			Unit: aws.String(data.Unit),
		}

		if err := datum.Validate(); err != nil {
			w.logger.Error(err, "invalid metric datum")
			continue
		}

		metricData = append(metricData, datum)
	}

	return metricData, nil
}
