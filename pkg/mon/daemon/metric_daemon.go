package daemon

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel/common"
	"github.com/applike/gosoline/pkg/mon"
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
	Dimensions mon.MetricDimensions
	Values     []float64
	Unit       string
}

type cwDaemon struct {
	sync.Mutex
	logger   mon.Logger
	settings *MetricSettings

	channel *metricChannel
	ticker  *time.Ticker
	writers []mon.MetricWriter

	batch          map[string]*BatchedMetricDatum
	defaults       map[string]*mon.MetricDatum
	dataPointCount int
}

type metricChannel struct {
	logger mon.Logger
	c      chan mon.MetricData
	closed bool
	lck    sync.RWMutex
}

func (c *metricChannel) write(batch mon.MetricData) {
	c.lck.RLock()
	defer c.lck.RUnlock()

	if c.closed {
		c.logger.Warnf("refusing to write %d metric datums to metric channel: channel is closed", len(batch))

		return
	}

	c.c <- batch
}

// Lock the channel metadata, close the channel and unlock it again.
// Why do we need a RW lock for the channel? Multiple possible choices:
//  - Just read until we get nothing more - does not work if a producer
//    writes more messages after we read "everything" to the channel. If
//    the producer writes enough messages, it could actually get stuck
//    because there is no consumer left and we only buffer 100 items
//  - Just add an (atomic) boolean flag: If we check whether we closed the
//    channel and then write to it, if not, we have a time-of-check to
//    time-of-use race condition. Between our check and writing to the
//    channel someone could have closed the channel.
//  - Just use recover when you get a panic: Would work, but this is really
//    not pretty.
func (c *metricChannel) close() {
	c.lck.Lock()
	defer c.lck.Unlock()

	close(c.c)
	c.closed = true
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
		channel: &metricChannel{
			c:      make(chan mon.MetricData, 100),
			closed: false,
		},
		defaults: make(map[string]*mon.MetricDatum, 0),
		settings: &MetricSettings{},
		writers:  make([]mon.MetricWriter, 0),
	}

	mon.InitializeMetricDaemon(cwDaemonContainer.instance)

	return cwDaemonContainer.instance
}

func (d *cwDaemon) GetType() string {
	return common.TypeBackground
}

func (d *cwDaemon) GetStage() int {
	return common.StageEssential
}

func (d *cwDaemon) Boot(config cfg.Config, logger mon.Logger) error {
	settings := getMetricSettings(config)

	writers := make([]mon.MetricWriter, len(settings.Writers))
	for i, t := range settings.Writers {
		writers[i] = ProvideMetricWriterByType(config, logger, t)
	}

	return d.BootWithInterfaces(logger, writers, settings)
}

func (d *cwDaemon) BootWithInterfaces(logger mon.Logger, writers []mon.MetricWriter, settings *MetricSettings) error {
	d.logger = logger.WithChannel("metrics")
	d.settings = settings
	d.writers = writers
	d.ticker = time.NewTicker(settings.Interval)
	d.channel.logger = logger.WithChannel("metrics-channel")

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

func (d *cwDaemon) IsEnabled() bool {
	return d.settings.Enabled
}

func (d *cwDaemon) AddDefaults(data ...*mon.MetricDatum) {
	d.Lock()
	defer d.Unlock()

	for _, datum := range data {
		id := datum.Id()
		d.defaults[id] = datum
	}
}

func (d *cwDaemon) Write(batch mon.MetricData) {
	d.channel.write(batch)
}

func (d *cwDaemon) emptyChannel() {
	d.channel.close()

	for data := range d.channel.c {
		d.appendBatch(data)
	}
}
func (d *cwDaemon) appendBatch(data mon.MetricData) {
	for _, dat := range data {
		d.append(dat)
	}
}

func (d *cwDaemon) append(datum *mon.MetricDatum) {
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

func (d *cwDaemon) amendFromDefault(datum *mon.MetricDatum) {
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

func (d *cwDaemon) buildMetricData() mon.MetricData {
	data := make([]*mon.MetricDatum, 0)

	for _, v := range d.batch {
		unit, value := d.calcValue(v.Unit, v.Values)

		datum := &mon.MetricDatum{
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
	case mon.UnitCountAverage:
		unit = mon.UnitCount
		value = average(values)
	case mon.UnitMilliseconds:
		value = average(values)
	case mon.UnitSeconds:
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
