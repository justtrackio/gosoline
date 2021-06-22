package metric

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel/common"
	"github.com/applike/gosoline/pkg/log"
	"sync"
	"time"
)

const defaultTimeFormat = "2006-01-02T15:04Z07:00"

type Settings struct {
	cfg.AppId
	Enabled  bool          `cfg:"enabled" default:"false"`
	Interval time.Duration `cfg:"interval" default:"60s"`
	Writers  []string      `cfg:"writers"`
}

func getMetricSettings(config cfg.Config) *Settings {
	settings := &Settings{}
	config.UnmarshalKey("metric", settings)

	settings.PadFromConfig(config)

	return settings
}

type BatchedMetricDatum struct {
	Priority   int
	Timestamp  time.Time
	MetricName string
	Dimensions Dimensions
	Values     []float64
	Unit       string
}

type Daemon struct {
	sync.Mutex
	logger   log.Logger
	settings *Settings

	channel *metricChannel
	ticker  *time.Ticker
	writers []Writer

	batch          map[string]*BatchedMetricDatum
	dataPointCount int
}

func NewDaemon(config cfg.Config, logger log.Logger) (*Daemon, error) {
	settings := getMetricSettings(config)

	channel := ProviderMetricChannel()
	channel.enabled = settings.Enabled
	channel.logger = logger.WithChannel("metrics")

	var err error
	var writers = make([]Writer, len(settings.Writers))

	for i, t := range settings.Writers {
		if writers[i], err = ProvideMetricWriterByType(config, logger, t); err != nil {
			return nil, fmt.Errorf("can not create metric writer of type %s: %w", t, err)
		}
	}

	return NewMetricDaemonWithInterfaces(logger, channel, writers, settings)
}

func NewMetricDaemonWithInterfaces(logger log.Logger, channel *metricChannel, writers []Writer, settings *Settings) (*Daemon, error) {
	return &Daemon{
		logger:         logger.WithChannel("metrics"),
		settings:       settings,
		channel:        channel,
		ticker:         time.NewTicker(settings.Interval),
		writers:        writers,
		batch:          make(map[string]*BatchedMetricDatum),
		dataPointCount: 0,
	}, nil
}

func (d *Daemon) IsEssential() bool {
	return false
}

func (d *Daemon) IsBackground() bool {
	return true
}

func (d *Daemon) GetStage() int {
	return common.StageEssential
}

func (d *Daemon) Run(ctx context.Context) error {
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
			d.emptyChannel()
			d.publish()
			return nil

		case data := <-d.channel.c:
			d.appendBatch(data)

		case <-d.ticker.C:
			d.publish()
		}
	}
}

func (d *Daemon) emptyChannel() {
	d.channel.close()

	for data := range d.channel.c {
		d.appendBatch(data)
	}
}

func (d *Daemon) appendBatch(data Data) {
	for _, dat := range data {
		d.append(dat)
	}
}

func (d *Daemon) append(datum *Datum) {
	d.dataPointCount++

	dimKey := datum.DimensionKey()
	timeKey := datum.Timestamp.Format(defaultTimeFormat)

	key := fmt.Sprintf("%s-%s-%s", datum.MetricName, dimKey, timeKey)

	if _, ok := d.batch[key]; !ok {
		d.amendFromDefault(datum)

		if err := datum.IsValid(); err != nil {
			d.logger.Warn("invalid metric: %s", err.Error())
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

func (d *Daemon) amendFromDefault(datum *Datum) {
	defId := datum.Id()
	def, ok := metricDefaults[defId]

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

func (d *Daemon) resetBatch() {
	d.batch = make(map[string]*BatchedMetricDatum)
	d.dataPointCount = 0

	for _, def := range metricDefaults {
		cpy := *def
		cpy.Timestamp = time.Now()

		d.append(&cpy)
	}
}

func (d *Daemon) publish() {
	size := len(d.batch)

	if size == 0 {
		return
	}

	data := d.buildMetricData()

	for _, w := range d.writers {
		w.Write(data)
	}

	d.logger.Info("published %d data points in %d metrics", d.dataPointCount, size)
	d.resetBatch()
}

func (d *Daemon) buildMetricData() Data {
	data := make([]*Datum, 0)

	for _, v := range d.batch {
		unit, value := d.calcValue(v.Unit, v.Values)

		datum := &Datum{
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

func (d *Daemon) calcValue(unit string, values []float64) (string, float64) {
	value := 0.0

	switch unit {
	case UnitCountAverage:
		unit = UnitCount
		value = average(values)
	case UnitMillisecondsAverage:
		unit = UnitMilliseconds
		value = average(values)
	case UnitSecondsAverage:
		unit = UnitSeconds
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
