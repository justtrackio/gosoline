package mon

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultTimeFormat = "2006-01-02T15:04Z07:00"

type CwDaemonSettings struct {
	Enabled bool
	Timeout time.Duration
}

type BatchedMetricDatum struct {
	Priority   int
	Timestamp  time.Time
	MetricName string
	Dimensions MetricDimensions
	Values     []float64
	Unit       string
}

type cwDaemon struct {
	sync.Mutex
	logger   Logger
	settings CwDaemonSettings

	channel chan *MetricDatum
	ticker  *time.Ticker
	writers []MetricWriter

	batch          map[string]*BatchedMetricDatum
	defaults       []*MetricDatum
	dataPointCount int
}

var cwDaemonContainer = struct {
	sync.Mutex
	instance *cwDaemon
}{}

func ProvideCwDaemon() *cwDaemon {
	cwDaemonContainer.Lock()
	defer cwDaemonContainer.Unlock()

	if cwDaemonContainer.instance != nil {
		return cwDaemonContainer.instance
	}

	cwDaemonContainer.instance = &cwDaemon{
		channel:  make(chan *MetricDatum, 100),
		defaults: make([]*MetricDatum, 0),
		writers:  make([]MetricWriter, 0),
	}

	return cwDaemonContainer.instance
}

func (d *cwDaemon) GetType() string {
	return "background"
}

func (d *cwDaemon) Boot(config cfg.Config, logger Logger) error {
	types := config.GetStringSlice("metric_writers")

	writers := make([]MetricWriter, len(types))
	for i, t := range types {
		writers[i] = ProvideMetricWriterByType(config, logger, t)
	}

	enabled := config.GetBool("metric_enabled")
	timeout := config.GetDuration("metric_daemon_timeout")

	return d.BootWithInterfaces(logger, writers, CwDaemonSettings{
		Enabled: enabled,
		Timeout: timeout,
	})
}

func (d *cwDaemon) BootWithInterfaces(logger Logger, writers []MetricWriter, settings CwDaemonSettings) error {
	d.logger = logger.WithChannel("metrics")
	d.settings = settings
	d.writers = writers
	d.ticker = time.NewTicker(settings.Timeout * time.Second)

	return nil
}

func (d *cwDaemon) Run(ctx context.Context) error {
	if !d.settings.Enabled {
		d.logger.Info("metrics not enabled..")
		return nil
	}

	d.resetBatch()

	for {
		select {
		case <-ctx.Done():
			d.settings.Enabled = false
			d.ticker.Stop()
			d.publish()
			return nil

		case dat := <-d.channel:
			d.append(dat)

		case <-d.ticker.C:
			d.publish()
		}
	}
}

func (d *cwDaemon) AddDefault(datum *MetricDatum) {
	d.defaults = append(d.defaults, datum)
}

func (d *cwDaemon) append(datum *MetricDatum) {
	d.dataPointCount++
	dims := make([]string, 0)

	for k, v := range datum.Dimensions {
		flat := fmt.Sprintf("%s:%s", k, v)
		dims = append(dims, flat)
	}

	sort.Strings(dims)
	dimKey := strings.Join(dims, "-")
	timeKey := datum.Timestamp.Format(defaultTimeFormat)

	key := fmt.Sprintf("%s-%s-%s", datum.MetricName, dimKey, timeKey)

	if _, ok := d.batch[key]; !ok {
		d.batch[key] = &BatchedMetricDatum{
			Priority:   datum.Priority,
			Timestamp:  datum.Timestamp,
			MetricName: datum.MetricName,
			Dimensions: datum.Dimensions,
			Unit:       datum.Unit,
			Values:     []float64{datum.Value},
		}
		return
	}

	existing := d.batch[key]
	existing.Values = append(existing.Values, datum.Value)
}

func (d *cwDaemon) resetBatch() {
	d.batch = make(map[string]*BatchedMetricDatum)
	d.dataPointCount = 0

	for _, def := range d.defaults {
		cpy := *def
		cpy.Timestamp = time.Now()

		d.append(&cpy)
	}
}

func (d *cwDaemon) publish() {
	size := len(d.batch)

	if size == 0 {
		return
	}

	data := d.buildMetricData()

	for _, w := range d.writers {
		w.Write(data)
	}

	d.logger.Infof("published %d data points in %d metrics", d.dataPointCount, size)
	d.resetBatch()
}

func (d *cwDaemon) buildMetricData() MetricData {
	data := make([]*MetricDatum, 0)

	for _, v := range d.batch {
		unit, value := d.calcValue(v.Unit, v.Values)

		datum := &MetricDatum{
			Priority:   v.Priority,
			Timestamp:  v.Timestamp,
			MetricName: v.MetricName,
			Dimensions: v.Dimensions,
			Unit:       unit,
			Value:      value,
		}

		data = append(data, datum)
	}

	return data
}

func (d *cwDaemon) calcValue(unit string, values []float64) (string, float64) {
	value := 0.0

	switch unit {
	case UnitCountAverage:
		unit = UnitCount
		value = average(values)
	case UnitMilliseconds:
		value = average(values)
	case UnitSeconds:
		value = average(values)
	default:
		value = sum(values)
	}

	return unit, value
}

func average(xs []float64) float64 {
	return sum(xs) / float64(len(xs))
}

func sum(xs []float64) float64 {
	total := 0.0

	for _, v := range xs {
		total += v
	}

	return total
}
