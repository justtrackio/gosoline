package metric

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/kernel/common"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	defaultTimeFormat       = "2006-01-02T15:04Z07:00"
	WriterTypeCloudwatch    = "cloudwatch"
	WriterTypeElasticsearch = "elasticsearch"
	WriterTypePrometheus    = "prometheus"
)

type WriterFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error)

var writerFactories = map[string]WriterFactory{}

func RegisterWriterFactory(name string, factory WriterFactory) {
	writerFactories[name] = factory
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
	kernel.EssentialBackgroundModule
	logger   log.Logger
	settings *Settings

	channel                 *metricChannel
	ticker                  *time.Ticker
	aggregatedMetricWriters []Writer
	rawMetricWriters        []Writer

	batch          map[string]*BatchedMetricDatum
	dataPointCount int

	errorThrottlesLck sync.Mutex
	errorThrottles    map[string]bool
}

func metricWriterAggrKey(typ string) string {
	return fmt.Sprintf("metric.writer_settings.%s.aggregate", typ)
}

func NewDaemonModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	settings, err := getMetricSettings(config)
	if err != nil {
		return nil, fmt.Errorf("could not get metric settings: %w", err)
	}

	channel := providerMetricChannel(func(channel *metricChannel) {
		channel.enabled = settings.Enabled
		channel.logger = logger.WithChannel("metrics")
	})

	if !settings.Enabled {
		return nil, nil
	}

	aggWriters := make([]Writer, 0)
	rawWriters := make([]Writer, 0)

	for _, typ := range settings.Writers {
		metricWriterAggrCnfKey := metricWriterAggrKey(typ)
		aggWriter, err := config.GetBool(metricWriterAggrCnfKey, false)
		if err != nil {
			return nil, fmt.Errorf("can not get bool from config at %s: %w", metricWriterAggrCnfKey, err)
		}

		factory, ok := writerFactories[typ]
		if !ok {
			return nil, fmt.Errorf("unrecognized writer type: %s", typ)
		}

		w, err := factory(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("could not create %s metric writer: %w", typ, err)
		}

		if aggWriter {
			aggWriters = append(aggWriters, w)
		} else {
			rawWriters = append(rawWriters, w)
		}
	}

	return NewMetricDaemonWithInterfaces(logger, channel, aggWriters, rawWriters, settings)
}

func NewMetricDaemonWithInterfaces(logger log.Logger, channel *metricChannel, aggWriters []Writer, rawWriters []Writer, settings *Settings) (kernel.Module, error) {
	return &Daemon{
		logger:                  logger.WithChannel("metrics"),
		settings:                settings,
		channel:                 channel,
		ticker:                  time.NewTicker(settings.Interval),
		aggregatedMetricWriters: aggWriters,
		rawMetricWriters:        rawWriters,
		batch:                   make(map[string]*BatchedMetricDatum),
		dataPointCount:          0,
		errorThrottles:          make(map[string]bool),
	}, nil
}

func (d *Daemon) GetStage() int {
	return common.StageEssential
}

func (d *Daemon) Run(ctx context.Context) error {
	d.resetBatch(ctx)

	// initialize the default metrics upon daemon module Run for raw writers
	metricDefaultsLock.Lock()
	d.rawFanout(ctx, funk.Values(metricDefaults))
	metricDefaultsLock.Unlock()

	for {
		select {
		case <-ctx.Done():
			d.ticker.Stop()
			d.emptyChannel(ctx)
			d.publish(ctx)

			return nil

		case <-d.channel.hasData:
			data := d.channel.read()
			d.rawFanout(ctx, data)
			d.appendBatch(ctx, data)

		case <-d.ticker.C:
			d.publish(ctx)
		}
	}
}

func (d *Daemon) emptyChannel(ctx context.Context) {
	d.channel.close()

	if data := d.channel.read(); len(data) > 0 {
		d.rawFanout(ctx, data)
		d.appendBatch(ctx, data)
	}
}

func (d *Daemon) rawFanout(ctx context.Context, data Data) {
	for _, w := range d.rawMetricWriters {
		w.Write(ctx, data)
	}
}

func (d *Daemon) appendBatch(ctx context.Context, data Data) {
	for _, dat := range data {
		d.append(ctx, dat)
	}
}

func (d *Daemon) append(ctx context.Context, datum *Datum) {
	d.dataPointCount++

	dimKV := datum.DimensionKV()
	timeKey := datum.Timestamp.Format(defaultTimeFormat)

	key := fmt.Sprintf("%s-%s-%s", datum.MetricName, dimKV, timeKey)

	if _, ok := d.batch[key]; !ok {
		amendFromDefault(datum)

		if err := datum.IsValid(); err != nil {
			if d.throttleError(err.Error()) {
				d.logger.Error(ctx, "invalid metric: %s", err.Error())
			}

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

func (d *Daemon) resetBatch(ctx context.Context) {
	d.batch = make(map[string]*BatchedMetricDatum)
	d.dataPointCount = 0

	for _, def := range metricDefaults {
		cpy := *def
		cpy.Timestamp = time.Now()

		d.append(ctx, &cpy)
	}
}

func (d *Daemon) publish(ctx context.Context) {
	size := len(d.batch)

	if size == 0 {
		return
	}

	data := d.buildMetricData()

	for _, w := range d.aggregatedMetricWriters {
		w.Write(ctx, data)
	}

	d.logger.Info(ctx, "published %d data points in %d metrics", d.dataPointCount, size)
	d.resetBatch(ctx)
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

// we don't want to log errors every time they occur - it is enough to log them once they occur, at least for a minute
func (d *Daemon) throttleError(err string) bool {
	d.errorThrottlesLck.Lock()
	defer d.errorThrottlesLck.Unlock()

	if d.errorThrottles[err] {
		return false
	}

	d.errorThrottles[err] = true

	// automatically unlock the err after a minute
	go func() {
		time.Sleep(time.Minute)

		d.errorThrottlesLck.Lock()
		defer d.errorThrottlesLck.Unlock()

		delete(d.errorThrottles, err)
	}()

	return true
}
