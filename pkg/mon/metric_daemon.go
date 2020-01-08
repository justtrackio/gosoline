package mon

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"sync"
	"time"
)

const defaultTimeFormat = "2006-01-02T15:04Z07:00"

type MetricSettings struct {
	cfg.AppId
	Enabled  bool          `cfg:"enabled" default:"false"`
	Interval time.Duration `cfg:"interval" default:"60s"`
	Writers  []string      `cfg:"writers"`
}

func getMetricSettings(config cfg.Config) *MetricSettings {
	settings := &MetricSettings{}
	config.UnmarshalKey("mon.metric", settings)

	settings.PadFromConfig(config)

	return settings
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
	settings *MetricSettings

	channel chan *MetricDatum
	ticker  *time.Ticker
	writers []MetricWriter

	batch          map[string]*BatchedMetricDatum
	defaults       map[string]*MetricDatum
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
		defaults: make(map[string]*MetricDatum, 0),
		settings: &MetricSettings{},
		writers:  make([]MetricWriter, 0),
	}

	return cwDaemonContainer.instance
}

func (d *cwDaemon) GetType() string {
	return "background"
}

func (d *cwDaemon) Boot(config cfg.Config, logger Logger) error {
	settings := getMetricSettings(config)

	writers := make([]MetricWriter, len(settings.Writers))
	for i, t := range settings.Writers {
		writers[i] = ProvideMetricWriterByType(config, logger, t)
	}

	return d.BootWithInterfaces(logger, writers, settings)
}

func (d *cwDaemon) BootWithInterfaces(logger Logger, writers []MetricWriter, settings *MetricSettings) error {
	d.logger = logger.WithChannel("metrics")
	d.settings = settings
	d.writers = writers
	d.ticker = time.NewTicker(settings.Interval)

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

func (d *cwDaemon) AddDefaults(data ...*MetricDatum) {
	d.Lock()
	defer d.Unlock()

	for _, datum := range data {
		id := datum.Id()
		d.defaults[id] = datum
	}
}

func (d *cwDaemon) append(datum *MetricDatum) {
	d.dataPointCount++

	dimKey := datum.DimensionKey()
	timeKey := datum.Timestamp.Format(defaultTimeFormat)

	key := fmt.Sprintf("%s-%s-%s", datum.MetricName, dimKey, timeKey)

	if _, ok := d.batch[key]; !ok {
		d.amendFromDefault(datum)

		if err := datum.IsValid(); err != nil {
			d.logger.Warnf("invalid metric: %s", err.Error())
			return
		}

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

func (d *cwDaemon) amendFromDefault(datum *MetricDatum) {
	defId := datum.Id()
	def, ok := d.defaults[defId]

	if !ok {
		return
	}

	if datum.Priority == 0 {
		datum.Priority = def.Priority
	}

	if datum.Unit == "" {
		datum.Unit = def.Unit
	}
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
