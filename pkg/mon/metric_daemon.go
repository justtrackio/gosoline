package mon

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel/common"
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

type MetricDaemon struct {
	sync.Mutex
	logger   Logger
	settings *MetricSettings

	channel *metricChannel
	ticker  *time.Ticker
	writers []MetricWriter

	batch          map[string]*BatchedMetricDatum
	dataPointCount int
}

func NewMetricDaemon(config cfg.Config, logger Logger) (*MetricDaemon, error) {
	settings := getMetricSettings(config)

	channel := ProviderMetricChannel()
	channel.enabled = settings.Enabled
	channel.logger = logger.WithChannel("metrics")

	var err error
	var writers = make([]MetricWriter, len(settings.Writers))

	for i, t := range settings.Writers {
		if writers[i], err = ProvideMetricWriterByType(config, logger, t); err != nil {
			return nil, fmt.Errorf("can not create metric writer of type %s: %w", t, err)
		}
	}

	return NewMetricDaemonWithInterfaces(logger, channel, writers, settings)
}

func NewMetricDaemonWithInterfaces(logger Logger, channel *metricChannel, writers []MetricWriter, settings *MetricSettings) (*MetricDaemon, error) {
	return &MetricDaemon{
		logger:         logger.WithChannel("metrics"),
		settings:       settings,
		channel:        channel,
		ticker:         time.NewTicker(settings.Interval),
		writers:        writers,
		batch:          make(map[string]*BatchedMetricDatum),
		dataPointCount: 0,
	}, nil
}

func (d *MetricDaemon) GetType() string {
	return common.TypeBackground
}

func (d *MetricDaemon) GetStage() int {
	return common.StageEssential
}

func (d *MetricDaemon) Run(ctx context.Context) error {
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

func (d *MetricDaemon) emptyChannel() {
	d.channel.close()

	for data := range d.channel.c {
		d.appendBatch(data)
	}
}

func (d *MetricDaemon) appendBatch(data MetricData) {
	for _, dat := range data {
		d.append(dat)
	}
}

func (d *MetricDaemon) append(datum *MetricDatum) {
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

func (d *MetricDaemon) amendFromDefault(datum *MetricDatum) {
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

func (d *MetricDaemon) resetBatch() {
	d.batch = make(map[string]*BatchedMetricDatum)
	d.dataPointCount = 0

	for _, def := range metricDefaults {
		cpy := *def
		cpy.Timestamp = time.Now()

		d.append(&cpy)
	}
}

func (d *MetricDaemon) publish() {
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

func (d *MetricDaemon) buildMetricData() MetricData {
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

func (d *MetricDaemon) calcValue(unit string, values []float64) (string, float64) {
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
