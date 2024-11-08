package metric

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	UnitPromCounter   StandardUnit = "prom-counter"
	UnitPromGauge     StandardUnit = "prom-gauge"
	UnitPromHistogram StandardUnit = "prom-histogram"
	UnitPromSummary   StandardUnit = "prom-summary"
)

type (
	prometheusWriterCtxKey string
	registryAppCtxKey      string
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
	promMetrics sync.Map
	registry    *prometheus.Registry
	namespace   string
	metricLimit int64
	metrics     *int64
}

func ProvidePrometheusWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	return appctx.Provide(ctx, prometheusWriterCtxKey("default"), func() (Writer, error) {
		return NewPrometheusWriter(ctx, config, logger)
	})
}

func NewPrometheusWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	settings := &PromSettings{}
	config.UnmarshalKey(promSettingsKey, settings)

	appId := cfg.GetAppIdFromConfig(config)
	namespace := promNSNamingStrategy(appId)

	registry, err := ProvideRegistry(ctx, "default")
	if err != nil {
		return nil, err
	}

	return NewPrometheusWriterWithInterfaces(logger, registry, namespace, settings.MetricLimit), nil
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

func (w *prometheusWriter) Write(batch Data) {
	if len(batch) == 0 {
		return
	}

	for _, datum := range batch {
		amendFromDefault(datum)

		if datum.Priority < w.GetPriority() {
			continue
		}

		w.promMetricFromDatum(datum)
	}

	w.logger.Debug("written %d metric data sets to prometheus", len(batch))
}

func (w *prometheusWriter) WriteOne(data *Datum) {
	w.Write(Data{data})
}

func (w *prometheusWriter) promMetricFromDatum(data *Datum) {
	defer func() {
		err := coffin.ResolveRecovery(recover())
		if err != nil {
			w.logger.Error("prom metric from datum for id %s: %w", data.Id(), err)
		}
	}()

	if strings.Contains(data.MetricName, "-") {
		w.logger.Warn("metric name %s is invalid, as it contains a - characters, gracefully replacing with an _ character", data.MetricName)
		data.MetricName = replacer.Replace(data.MetricName)
	}

	switch data.Unit {
	case UnitCount:
		fallthrough
	case UnitPromCounter:
		w.promCounter(data)
	case UnitPromSummary:
		fallthrough
	case UnitMilliseconds:
		fallthrough
	case UnitSeconds:
		w.promSummary(data)
	case UnitPromHistogram:
		w.promHistogram(data)
	default:
		w.promGauge(data)
	}
}

func (w *prometheusWriter) buildHelp(data *Datum) string {
	return fmt.Sprintf("unit: %s", data.Unit)
}

func (w *prometheusWriter) createCounter(datum *Datum) *prometheus.CounterVec {
	return promauto.With(w.registry).NewCounterVec(prometheus.CounterOpts{
		Namespace: w.namespace,
		Name:      datum.MetricName,
		Help:      w.buildHelp(datum),
	}, datum.DimensionKeys())
}

func (w *prometheusWriter) createGauge(datum *Datum) *prometheus.GaugeVec {
	return promauto.With(w.registry).NewGaugeVec(prometheus.GaugeOpts{
		Namespace: w.namespace,
		Name:      datum.MetricName,
		Help:      w.buildHelp(datum),
	}, datum.DimensionKeys())
}

func (w *prometheusWriter) createSummary(datum *Datum) *prometheus.SummaryVec {
	return promauto.With(w.registry).NewSummaryVec(prometheus.SummaryOpts{
		Namespace: w.namespace,
		Name:      datum.MetricName,
		Help:      w.buildHelp(datum),
	}, datum.DimensionKeys())
}

func (w *prometheusWriter) createHistogram(datum *Datum) *prometheus.HistogramVec {
	return promauto.With(w.registry).NewHistogramVec(prometheus.HistogramOpts{
		Namespace: w.namespace,
		Name:      datum.MetricName,
		Help:      w.buildHelp(datum),
	}, datum.DimensionKeys())
}

func (w *prometheusWriter) addMetric(id string, metric any) error {
	if atomic.LoadInt64(w.metrics) >= w.metricLimit {
		w.logger.Error("fail to write metric due to exceeding limit")

		return errors.New("metric limit exceeded")
	}

	w.promMetrics.Store(id, metric)
	atomic.AddInt64(w.metrics, 1)

	return nil
}

func (w *prometheusWriter) promCounter(datum *Datum) {
	id := datum.Id()

	metricI, ok := w.promMetrics.Load(id)
	if !ok {
		var err error
		metric := w.createCounter(datum)

		err = w.addMetric(id, metric)
		if err != nil {
			return // error is logged in w.addMetric already
		}

		metricI = metric
	}

	metric := metricI.(*prometheus.CounterVec)
	metric.
		With(prometheus.Labels(datum.Dimensions)).
		Add(datum.Value)
}

func (w *prometheusWriter) promGauge(datum *Datum) {
	id := datum.Id()
	metricI, ok := w.promMetrics.Load(id)
	if !ok {
		var err error
		metric := w.createGauge(datum)

		err = w.addMetric(id, metric)
		if err != nil {
			return // error is logged in w.addMetric already
		}

		metricI = metric
	}

	metric := metricI.(*prometheus.GaugeVec)
	metric.
		With(prometheus.Labels(datum.Dimensions)).
		Set(datum.Value)
}

func (w *prometheusWriter) promSummary(datum *Datum) {
	id := datum.Id()

	metricI, ok := w.promMetrics.Load(id)
	if !ok {
		var err error
		metric := w.createSummary(datum)

		err = w.addMetric(id, metric)
		if err != nil {
			return // error is logged in w.addMetric already
		}

		metricI = metric
	}

	metric := metricI.(*prometheus.SummaryVec)
	metric.
		With(prometheus.Labels(datum.Dimensions)).
		Observe(datum.Value)
}

func (w *prometheusWriter) promHistogram(datum *Datum) {
	id := datum.Id()

	metricI, ok := w.promMetrics.Load(id)
	if !ok {
		var err error
		metric := w.createHistogram(datum)

		err = w.addMetric(id, metric)
		if err != nil {
			return // error is logged in w.addMetric already
		}

		metricI = metric
	}

	metric := metricI.(*prometheus.HistogramVec)
	metric.
		With(prometheus.Labels(datum.Dimensions)).
		Observe(datum.Value)
}
