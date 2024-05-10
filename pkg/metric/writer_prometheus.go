package metric

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func init() {
	RegisterWriterFactory(WriterTypePrometheus, ProvidePrometheusWriter)
}

var _ Writer = &prometheusWriter{}

const (
	errFailedToRegisterMetricMsg                  = "register metric %s: %w"
	errFailedToAddMetricToPrometheus              = "add metric: %w"
	UnitPromCounter                  StandardUnit = "prom-counter"
	UnitPromGauge                    StandardUnit = "prom-gauge"
	UnitPromHistogram                StandardUnit = "prom-histogram"
	UnitPromSummary                  StandardUnit = "prom-summary"
)

type (
	prometheusMetricProcessor func(metric prometheus.Collector)
	prometheusWriterCtxKey    string
	registryAppCtxKey         string
)

func ProvideRegistry(ctx context.Context, name string) (*prometheus.Registry, error) {
	return appctx.Provide(ctx, registryAppCtxKey(name), func() (*prometheus.Registry, error) {
		registry := prometheus.NewRegistry()
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		registry.MustRegister(collectors.NewGoCollector())

		return registry, nil
	})
}

type prometheusWriter struct {
	logger      log.Logger
	registry    *prometheus.Registry
	namespace   string
	metricLimit int64
	metrics     *int64
}

// ProvidePrometheusWriter provides a prometheus writer. If one is registered under the default key in
// the appctx, this is returned, else creates a new one and registers it.
// For more information on the prometheus writer itself see [NewPrometheusWriter].
func ProvidePrometheusWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	return appctx.Provide(ctx, prometheusWriterCtxKey("default"), func() (Writer, error) {
		return NewPrometheusWriter(ctx, config, logger)
	})
}

// NewPrometheusWriter creates a new prometheus metric writer.
// The prometheus writer writes metrics provided to it to the go prometheus client library.
// Metrics with Kind "total" are dropped before writing them.
func NewPrometheusWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	promSettings := &PrometheusSettings{}
	if err := getMetricWriterSettings(config, WriterTypePrometheus, promSettings); err != nil {
		return nil, fmt.Errorf("could not get prometheus writer settings: %w", err)
	}

	appId := cfg.GetAppIdFromConfig(config)
	namespace := promNSNamingStrategy(appId)

	registry, err := ProvideRegistry(ctx, prometheusDefaultRegistry)
	if err != nil {
		return nil, err
	}

	return NewPrometheusWriterWithInterfaces(logger, registry, namespace, promSettings.MetricLimit), nil
}

func NewPrometheusWriterWithInterfaces(logger log.Logger, registry *prometheus.Registry, namespace string, metricLimit int64) Writer {
	return &prometheusWriter{
		logger:      logger.WithChannel("metrics"),
		registry:    registry,
		namespace:   namespace,
		metricLimit: metricLimit,
		metrics:     mdl.Box(int64(0)),
	}
}

func (w *prometheusWriter) GetPriority() int {
	return PriorityLow
}

// shouldFilterMetric drops all metrics which are not passing the configured priority threshold.
// For cloudwatch (since it does not support summing up over all dimensions for a metric) we are
// writing total metrics. A total metric is a single timeseries within a metric that has dimensions
// for other timeseries, such that we can visualize / use the total of everything that we are measuring
// with that metric. This is not needed on prometheus, as we can just create the total through summing
// up all our dimensions.
func (w *prometheusWriter) shouldFilterMetric(datum *Datum) bool {
	return datum.Priority < w.GetPriority() || datum.Kind == KindTotal
}

func (w *prometheusWriter) Write(batch Data) {
	if len(batch) == 0 {
		return
	}

	for _, datum := range batch {
		datum = mdl.Box(preprocessPrometheusMetric(datum))
		amendFromDefault(datum)

		if w.shouldFilterMetric(datum) {
			continue
		}

		w.writeMetricFromDatum(datum)
	}

	w.logger.Debug("written %d metric data sets to prometheus", len(batch))
}

func (w *prometheusWriter) WriteOne(data *Datum) {
	w.Write(Data{data})
}

func (w *prometheusWriter) writeMetricFromDatum(datum *Datum) {
	defer func() {
		err := coffin.ResolveRecovery(recover())
		if err != nil {
			w.logger.Error("writing prometheus metric from datum for id %s: %w", w.DatumId(datum), err)
		}
	}()

	if strings.Contains(datum.MetricName, "-") {
		w.logger.Error("metric name %s is invalid, as it contains a - characters, gracefully replacing with an _ character", datum.MetricName)
		datum.MetricName = replacer.Replace(datum.MetricName)
	}

	switch datum.Unit {
	case UnitCount, UnitPromCounter:
		w.counter(datum)
	case UnitPromSummary, UnitMilliseconds, UnitSeconds:
		w.summary(datum)
	case UnitPromHistogram:
		w.histogram(datum)
	default:
		w.gauge(datum)
	}
}

func (w *prometheusWriter) buildHelp(data *Datum) string {
	return fmt.Sprintf("unit: %s", data.Unit)
}

func (w *prometheusWriter) createCounter(datum *Datum) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: w.namespace,
		Name:      datum.MetricName,
		Help:      w.buildHelp(datum),
	}, w.DatumDimensionKeys(datum))
}

func (w *prometheusWriter) createGauge(datum *Datum) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: w.namespace,
		Name:      datum.MetricName,
		Help:      w.buildHelp(datum),
	}, w.DatumDimensionKeys(datum))
}

func (w *prometheusWriter) createSummary(datum *Datum) *prometheus.SummaryVec {
	return prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Namespace: w.namespace,
		Name:      datum.MetricName,
		Help:      w.buildHelp(datum),
	}, w.DatumDimensionKeys(datum))
}

func (w *prometheusWriter) createHistogram(datum *Datum) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: w.namespace,
		Name:      datum.MetricName,
		Help:      w.buildHelp(datum),
	}, w.DatumDimensionKeys(datum))
}

func (w *prometheusWriter) addMetric() error {
	if atomic.LoadInt64(w.metrics) >= w.metricLimit {
		return errors.New("metric limit exceeded")
	}

	atomic.AddInt64(w.metrics, 1)

	return nil
}

func handleRegistrationError(err error) (prometheus.Collector, error) {
	are := &prometheus.AlreadyRegisteredError{}
	if errors.As(err, are) {
		return are.ExistingCollector, nil
	} else {
		return nil, err
	}
}

func (w *prometheusWriter) registerAndProcessMetric(metric prometheus.Collector, metricName string, processorFn prometheusMetricProcessor) error {
	if err := w.registry.Register(metric); err != nil {
		metricR, err := handleRegistrationError(err)
		if err != nil {
			return fmt.Errorf(errFailedToRegisterMetricMsg, metricName, err)
		}

		metric = metricR
	} else {
		err = w.addMetric()
		if err != nil {
			return fmt.Errorf(errFailedToAddMetricToPrometheus, err)
		}
	}

	processorFn(metric)

	return nil
}

func (w *prometheusWriter) counter(datum *Datum) {
	metric := w.createCounter(datum)

	err := w.registerAndProcessMetric(metric, datum.MetricName, func(metric prometheus.Collector) {
		metric.(*prometheus.CounterVec).
			With(prometheus.Labels(datum.Dimensions)).
			Add(datum.Value)
	})
	if err != nil {
		w.logger.Error("writing prometheus counter for datum %s: %v", datum.MetricName, err)
	}
}

func (w *prometheusWriter) gauge(datum *Datum) {
	metric := w.createGauge(datum)

	err := w.registerAndProcessMetric(metric, datum.MetricName, func(metric prometheus.Collector) {
		metric.(*prometheus.GaugeVec).
			With(prometheus.Labels(datum.Dimensions)).
			Set(datum.Value)
	})
	if err != nil {
		w.logger.Error("writing prometheus gauge for datum %s: %v", datum.MetricName, err)
	}
}

func (w *prometheusWriter) summary(datum *Datum) {
	metric := w.createSummary(datum)

	err := w.registerAndProcessMetric(metric, datum.MetricName, func(metric prometheus.Collector) {
		metric.(*prometheus.SummaryVec).
			With(prometheus.Labels(datum.Dimensions)).
			Observe(datum.Value)
	})
	if err != nil {
		w.logger.Error("writing prometheus summary for datum %s: %v", datum.MetricName, err)
	}
}

func (w *prometheusWriter) histogram(datum *Datum) {
	metric := w.createHistogram(datum)

	err := w.registerAndProcessMetric(metric, datum.MetricName, func(metric prometheus.Collector) {
		metric.(*prometheus.HistogramVec).
			With(prometheus.Labels(datum.Dimensions)).
			Observe(datum.Value)
	})
	if err != nil {
		w.logger.Error("writing prometheus histogram for datum %s: %v", datum.MetricName, err)
	}
}

func (w *prometheusWriter) DatumId(datum *Datum) string {
	return fmt.Sprintf("%s:%v", datum.MetricName, w.DatumDimensionKeys(datum))
}

func (w *prometheusWriter) DatumDimensionKeys(datum *Datum) []string {
	dims := make([]string, 0)

	for k := range datum.Dimensions {
		dims = append(dims, k)
	}

	sort.Slice(dims, func(i, j int) bool {
		return dims[i] > dims[j]
	})

	return dims
}

func preprocessPrometheusMetric(datum *Datum) Datum {
	d := *datum
	d.Dimensions = maps.Clone(datum.Dimensions)

	for dimension, value := range d.Dimensions {
		if value == DimensionDefault {
			d.Dimensions[dimension] = ""
		}
	}

	return d
}
