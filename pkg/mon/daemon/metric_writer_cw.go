package daemon

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/cloudwatch/cloudwatchiface"
	"github.com/jonboulle/clockwork"
	"time"
)

const (
	chunkSizeCloudWatch = 20
	minusOneWeek        = -1 * 7 * 24 * time.Hour
	plusOneHour         = 1 * time.Hour
)

type cwWriter struct {
	logger   mon.Logger
	clock    clockwork.Clock
	cw       cloudwatchiface.CloudWatchAPI
	executor cloud.RequestExecutor
	settings *MetricSettings
}

func NewMetricCwWriter(config cfg.Config, logger mon.Logger) *cwWriter {
	settings := getMetricSettings(config)

	clock := clockwork.NewRealClock()
	cw := mon.ProvideCloudWatchClient(config)
	executor := cloud.NewBackoffExecutor(logger, &cloud.BackoffResource{
		Type: "cloud",
		Name: "watch",
	}, &cloud.BackoffSettings{
		Enabled:             true,
		Blocking:            false,
		CancelDelay:         1 * time.Second,
		InitialInterval:     50 * time.Millisecond,
		RandomizationFactor: 0.5,
		Multiplier:          1.5,
		MaxInterval:         10 * time.Second,
		MaxElapsedTime:      15 * time.Minute,
		MetricEnabled:       false,
	})

	return NewMetricCwWriterWithInterfaces(logger, clock, cw, executor, settings)
}

func NewMetricCwWriterWithInterfaces(logger mon.Logger, clock clockwork.Clock, cw cloudwatchiface.CloudWatchAPI, executor cloud.RequestExecutor, settings *MetricSettings) *cwWriter {
	return &cwWriter{
		logger:   logger.WithChannel("metrics"),
		clock:    clock,
		cw:       cw,
		executor: executor,
		settings: settings,
	}
}

func (w *cwWriter) GetPriority() int {
	return mon.PriorityHigh
}

func (w *cwWriter) WriteOne(data *mon.MetricDatum) {
	w.Write(mon.MetricData{data})
}

func (w *cwWriter) Write(batch mon.MetricData) {
	if !w.settings.Enabled || len(batch) == 0 {
		return
	}

	metricData, err := w.buildMetricData(batch)
	namespace := fmt.Sprintf("%s/%s/%s/%s", w.settings.Project, w.settings.Environment, w.settings.Family, w.settings.Application)

	if err != nil {
		w.logger.WithFields(mon.Fields{
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

		_, err := w.executor.Execute(context.Background(), func() (request *request.Request, i interface{}) {
			return w.cw.PutMetricDataRequest(&input)
		})

		if err != nil {
			w.logger.Error(err, "could not write metric data")
			continue
		}
	}

	w.logger.Debugf("written %d metric data sets to cloudwatch", len(metricData))
}

func (w *cwWriter) buildMetricData(batch mon.MetricData) ([]*cloudwatch.MetricDatum, error) {
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
