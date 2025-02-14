package metric

import (
	"context"
	"errors"
	"fmt"
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

const (
	errFailedToRegisterMetricMsg                         = "fail to register metric %s: %w"
	errFailedToAddMetricToPrometheusCounter              = "fail to add metric to prometheus counter: %w"
	UnitPromCounter                         StandardUnit = "prom-counter"
	UnitPromGauge                           StandardUnit = "prom-gauge"
	UnitPromHistogram                       StandardUnit = "prom-histogram"
	UnitPromSummary                         StandardUnit = "prom-summary"
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
	settings := getMetricSettings(config)

	appId := cfg.GetAppIdFromConfig(config)
	namespace := promNSNamingStrategy(appId)

	registry, err := ProvideRegistry(ctx, prometheusDefaultRegistry)
	if err != nil {
		return nil, err
	}

	return NewPrometheusWriterWithInterfaces(logger, registry, namespace, settings.WriterSettings.Prometheus.MetricLimit), nil
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

func (w *prometheusWriter) promMetricFromDatum(datum *Datum) {
	defer func() {
		err := coffin.ResolveRecovery(recover())
		if err != nil {
			w.logger.Error("writing prometheus metric from datum for id %s: %w", w.DatumId(datum), err)
		}
	}()

	if strings.Contains(datum.MetricName, "-") {
		w.logger.Warn("metric name %s is invalid, as it contains a - characters, gracefully replacing with an _ character", datum.MetricName)
		datum.MetricName = replacer.Replace(datum.MetricName)
	}

	switch datum.Unit {
	case UnitCount:
		fallthrough
	case UnitPromCounter:
		w.promCounter(datum)
	case UnitPromSummary:
		fallthrough
	case UnitMilliseconds:
		fallthrough
	case UnitSeconds:
		w.promSummary(datum)
	case UnitPromHistogram:
		w.promHistogram(datum)
	default:
		w.promGauge(datum)
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

func (w *prometheusWriter) promCounter(datum *Datum) {
	metric := w.createCounter(datum)

	if err := w.registry.Register(metric); err != nil {
		are := &prometheus.AlreadyRegisteredError{}
		if errors.As(err, are) {
			metric = are.ExistingCollector.(*prometheus.CounterVec)
		} else {
			w.logger.Error(errFailedToRegisterMetricMsg, datum.MetricName, err)
		}
	} else {
		err = w.addMetric()
		if err != nil {
			w.logger.Error(errFailedToAddMetricToPrometheusCounter, err)

			return
		}
	}

	metric.
		With(prometheus.Labels(datum.Dimensions)).
		Add(datum.Value)
}

func (w *prometheusWriter) promGauge(datum *Datum) {
	metric := w.createGauge(datum)

	if err := w.registry.Register(metric); err != nil {
		are := &prometheus.AlreadyRegisteredError{}
		if errors.As(err, are) {
			metric = are.ExistingCollector.(*prometheus.GaugeVec)
		} else {
			w.logger.Error(errFailedToRegisterMetricMsg, datum.MetricName, err)
		}
	} else {
		err = w.addMetric()
		if err != nil {
			w.logger.Error(errFailedToAddMetricToPrometheusCounter, err)

			return
		}
	}

	metric.
		With(prometheus.Labels(datum.Dimensions)).
		Set(datum.Value)
}

func (w *prometheusWriter) promSummary(datum *Datum) {
	metric := w.createSummary(datum)

	if err := w.registry.Register(metric); err != nil {
		are := &prometheus.AlreadyRegisteredError{}
		if errors.As(err, are) {
			metric = are.ExistingCollector.(*prometheus.SummaryVec)
		} else {
			w.logger.Error(errFailedToRegisterMetricMsg, datum.MetricName, err)
		}
	} else {
		err = w.addMetric()
		if err != nil {
			w.logger.Error(errFailedToAddMetricToPrometheusCounter, err)

			return
		}
	}

	metric.
		With(prometheus.Labels(datum.Dimensions)).
		Observe(datum.Value)
}

func (w *prometheusWriter) promHistogram(datum *Datum) {
	metric := w.createHistogram(datum)

	if err := w.registry.Register(metric); err != nil {
		are := &prometheus.AlreadyRegisteredError{}
		if errors.As(err, are) {
			metric = are.ExistingCollector.(*prometheus.HistogramVec)
		} else {
			w.logger.Error(errFailedToRegisterMetricMsg, datum.MetricName, err)
		}
	} else {
		err = w.addMetric()
		if err != nil {
			w.logger.Error(errFailedToAddMetricToPrometheusCounter, err)

			return
		}
	}

	metric.
		With(prometheus.Labels(datum.Dimensions)).
		Observe(datum.Value)
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
