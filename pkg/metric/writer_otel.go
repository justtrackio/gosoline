package metric

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const (
	// WriterTypeOtel pushes metrics to an OTEL collector via OTLP, applying OTEL semantic-convention
	// naming and UCUM units. Identity is carried as resource attributes, not in the metric name.
	WriterTypeOtel = "otel"

	otelInstrumentationName = "github.com/justtrackio/gosoline/pkg/metric"
)

func init() {
	RegisterWriterFactory(WriterTypeOtel, ProvideOtelWriter)
}

var _ Writer = &otelWriter{}

type OtelWriterSettings struct {
	// Aggregate controls whether this writer receives daemon-aggregated batches (true) or raw data points (false).
	Aggregate bool `cfg:"aggregate" default:"false"`
	// Interval is the OTLP push (PeriodicReader) export interval.
	Interval time.Duration `cfg:"interval" default:"15s"`
}

type otelWriterCtxKey string

type otelWriter struct {
	logger log.Logger
	meter  otelmetric.Meter

	lck        sync.Mutex
	counters   map[string]otelmetric.Float64Counter
	gauges     map[string]otelmetric.Float64Gauge
	histograms map[string]otelmetric.Float64Histogram
}

// ProvideOtelWriter provides a shared OTLP metric writer from the app context.
func ProvideOtelWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	return appctx.Provide(ctx, otelWriterCtxKey("default"), func() (Writer, error) {
		return NewOtelWriter(ctx, config, logger)
	})
}

// NewOtelWriter builds an OTLP metric writer backed by an OTEL MeterProvider with a PeriodicReader.
// The MeterProvider uses the shared OTEL resource so metrics correlate with traces and logs.
func NewOtelWriter(ctx context.Context, config cfg.Config, logger log.Logger) (Writer, error) {
	writerSettings, err := getMetricWriterSettings[OtelWriterSettings](config, WriterTypeOtel)
	if err != nil {
		return nil, fmt.Errorf("could not get otel writer settings: %w", err)
	}

	otelSettings, err := otel.ReadSettings(config)
	if err != nil {
		return nil, err
	}

	res, err := otel.BuildResource(config, otelSettings.Resource)
	if err != nil {
		return nil, fmt.Errorf("could not build otel resource: %w", err)
	}

	exporter, err := otel.BuildMetricExporter(ctx, otelSettings.Exporter)
	if err != nil {
		return nil, fmt.Errorf("could not build otel metric exporter: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(writerSettings.Interval))
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)

	otel.Register(otel.PriorityMetrics, provider)

	return NewOtelWriterWithInterfaces(logger, provider.Meter(otelInstrumentationName)), nil
}

func NewOtelWriterWithInterfaces(logger log.Logger, meter otelmetric.Meter) Writer {
	return &otelWriter{
		logger:     logger.WithChannel("metrics"),
		meter:      meter,
		counters:   make(map[string]otelmetric.Float64Counter),
		gauges:     make(map[string]otelmetric.Float64Gauge),
		histograms: make(map[string]otelmetric.Float64Histogram),
	}
}

func (w *otelWriter) GetPriority() int {
	return PriorityLow
}

func (w *otelWriter) WriteOne(ctx context.Context, data *Datum) {
	w.Write(ctx, Data{data})
}

func (w *otelWriter) Write(ctx context.Context, batch Data) {
	for _, datum := range batch {
		if datum == nil {
			continue
		}

		amendFromDefault(datum)

		// total metrics exist only to support CloudWatch cross-dimension sums; not needed for OTEL.
		if datum.Kind.kind == KindTotal.kind {
			continue
		}

		if err := w.record(ctx, datum); err != nil {
			w.logger.Error(ctx, "could not write otel metric %s: %w", datum.MetricName, err)
		}
	}
}

func (w *otelWriter) record(ctx context.Context, datum *Datum) error {
	name := FormatOtelMetricName(datum.MetricName)
	unit := ToUcumUnit(datum.Unit)
	attrs := otelmetric.WithAttributes(w.attributes(datum)...)

	switch w.effectiveKind(datum) {
	case kindCounter:
		instrument, err := w.counter(name, unit, datum.Kind.help)
		if err != nil {
			return err
		}
		instrument.Add(ctx, datum.Value, attrs)
	case kindHistogram, kindSummary:
		instrument, err := w.histogram(name, unit, datum.Kind.help)
		if err != nil {
			return err
		}
		instrument.Record(ctx, datum.Value, attrs)
	default:
		instrument, err := w.gauge(name, unit, datum.Kind.help)
		if err != nil {
			return err
		}
		instrument.Record(ctx, datum.Value, attrs)
	}

	return nil
}

// effectiveKind resolves the metric kind, falling back to the unit when no explicit kind is set,
// mirroring the prometheus writer so both writers classify metrics consistently.
func (w *otelWriter) effectiveKind(datum *Datum) kind {
	switch datum.Kind.kind {
	case kindCounter, kindGauge, kindHistogram, kindSummary:
		return datum.Kind.kind
	}

	switch datum.Unit {
	case UnitCount:
		return kindCounter
	case UnitMilliseconds, UnitSeconds:
		return kindHistogram
	default:
		return kindGauge
	}
}

func (w *otelWriter) attributes(datum *Datum) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(datum.Dimensions))

	for key, value := range datum.Dimensions {
		if value == DimensionDefault {
			continue
		}

		attrs = append(attrs, attribute.String(key, value))
	}

	return attrs
}

func (w *otelWriter) counter(name, unit, help string) (otelmetric.Float64Counter, error) {
	w.lck.Lock()
	defer w.lck.Unlock()

	if instrument, ok := w.counters[name]; ok {
		return instrument, nil
	}

	instrument, err := w.meter.Float64Counter(name, otelmetric.WithUnit(unit), otelmetric.WithDescription(help))
	if err != nil {
		return nil, err
	}

	w.counters[name] = instrument

	return instrument, nil
}

func (w *otelWriter) gauge(name, unit, help string) (otelmetric.Float64Gauge, error) {
	w.lck.Lock()
	defer w.lck.Unlock()

	if instrument, ok := w.gauges[name]; ok {
		return instrument, nil
	}

	instrument, err := w.meter.Float64Gauge(name, otelmetric.WithUnit(unit), otelmetric.WithDescription(help))
	if err != nil {
		return nil, err
	}

	w.gauges[name] = instrument

	return instrument, nil
}

func (w *otelWriter) histogram(name, unit, help string) (otelmetric.Float64Histogram, error) {
	w.lck.Lock()
	defer w.lck.Unlock()

	if instrument, ok := w.histograms[name]; ok {
		return instrument, nil
	}

	instrument, err := w.meter.Float64Histogram(name, otelmetric.WithUnit(unit), otelmetric.WithDescription(help))
	if err != nil {
		return nil, err
	}

	w.histograms[name] = instrument

	return instrument, nil
}
