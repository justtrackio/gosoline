package metric

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel/common"
	"github.com/justtrackio/gosoline/pkg/log"
)

const defaultTimeFormat = "2006-01-02T15:04Z07:00"

type NamingSettings struct {
	Pattern string `cfg:"pattern,nodecode" default:"{project}/{env}/{family}/{group}-{app}"`
}

type Cloudwatch struct {
	Naming NamingSettings `cfg:"naming"`
}

type Settings struct {
	cfg.AppId
	Enabled    bool          `cfg:"enabled" default:"false"`
	Interval   time.Duration `cfg:"interval" default:"60s"`
	Cloudwatch Cloudwatch    `cfg:"cloudwatch"`
	Writer     string        `cfg:"writer"`
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
	Unit       types.StandardUnit
}

type Daemon struct {
	sync.Mutex
	logger   log.Logger
	settings *Settings

	channel *metricChannel
	ticker  *time.Ticker
	writer  Writer

	batch          map[string]*BatchedMetricDatum
	dataPointCount int
}

func NewDaemon(ctx context.Context, config cfg.Config, logger log.Logger) (*Daemon, error) {
	var metricWriter Writer
	var err error

	settings := getMetricSettings(config)

	channel := ProviderMetricChannel()
	channel.enabled = settings.Enabled
	channel.logger = logger.WithChannel("metrics")

	if settings.Enabled {
		metricWriter, err = ProvideMetricWriterByType(ctx, config, logger, settings.Writer)
		if err != nil {
			return nil, fmt.Errorf("can not create metric writer of type %s: %w", settings.Writer, err)
		}
	}

	return NewMetricDaemonWithInterfaces(logger, channel, metricWriter, settings)
}

func NewMetricDaemonWithInterfaces(logger log.Logger, channel *metricChannel, writer Writer, settings *Settings) (*Daemon, error) {
	return &Daemon{
		logger:         logger.WithChannel("metrics"),
		settings:       settings,
		channel:        channel,
		ticker:         time.NewTicker(settings.Interval),
		writer:         writer,
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
		amendFromDefault(datum)

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

	d.writer.Write(data)

	d.logger.Info("published %d data points in %d metrics", d.dataPointCount, size)
	d.resetBatch()
}

func (d *Daemon) buildMetricData() Data {
	data := make([]*Datum, 0)

	for _, v := range d.batch {
		unit, value := resolveCustomUnit(v.Unit, v.Values)

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
