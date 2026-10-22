package kvstore

import (
	"context"
	"errors"
	"testing"
	"time"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type failingGetBatchStore struct {
	KvStore[int]
	err   error
	calls int
	keys  []any
}

func (s *failingGetBatchStore) GetBatch(_ context.Context, keys any, _ any) ([]any, error) {
	s.calls++
	s.keys = keys.([]any)

	return nil, s.err
}

type namespacedMetricWriter struct {
	metric.Writer
	namespace string
}

func (w namespacedMetricWriter) Write(ctx context.Context, data metric.Data) {
	for _, datum := range data {
		if datum.Namespace == "" {
			datum.Namespace = w.namespace
		}
	}

	w.Writer.Write(ctx, data)
}

func (w namespacedMetricWriter) WriteOne(ctx context.Context, datum *metric.Datum) {
	if datum.Namespace == "" {
		datum.Namespace = w.namespace
	}

	w.Writer.WriteOne(ctx, datum)
}

type kvCounterKey struct {
	model string
	store string
	hit   string
}

func gatherKvCounters(t *testing.T, registry *prometheus.Registry, name string) map[kvCounterKey]float64 {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)

	var counterFamily *dto.MetricFamily
	for _, family := range families {
		if family.GetName() == name {
			counterFamily = family

			break
		}
	}
	require.NotNil(t, counterFamily)
	require.Equal(t, dto.MetricType_COUNTER, counterFamily.GetType())

	values := make(map[kvCounterKey]float64, len(counterFamily.GetMetric()))
	for _, sample := range counterFamily.GetMetric() {
		labels := make(map[string]string, len(sample.GetLabel()))
		for _, label := range sample.GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}

		values[kvCounterKey{
			model: labels["model_id"],
			store: labels["store_type"],
			hit:   labels["hit"],
		}] = sample.GetCounter().GetValue()
	}

	return values
}

func TestMetricStore_GetBatchCountsFailedDelegatedKeysAndNotConversionFailures(t *testing.T) {
	const (
		model     = "test-model"
		storeName = "test-store"
	)

	registry := prometheus.NewRegistry()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	exporter := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, time.Second)
	writer := namespacedMetricWriter{Writer: exporter, namespace: metricNamespace}
	writer.Write(t.Context(), getDefaultMetrics(model, storeName))

	storeErr := errors.New("read failed")
	store := &failingGetBatchStore{err: storeErr}
	wrapped := &MetricStore[int]{
		KvStore:      store,
		metricWriter: writer,
		model:        model,
		store:        storeName,
	}

	_, err := wrapped.GetBatch(t.Context(), 42, nil)
	require.Error(t, err)
	require.Zero(t, store.calls)
	attemptedValues := gatherKvCounters(t, registry, "gosoline_kvstore_attempted_reads_total")
	require.Equal(t, map[kvCounterKey]float64{{model: model, store: storeName}: 0.0}, attemptedValues)

	_, err = wrapped.GetBatch(t.Context(), []any{"a", "b", "c"}, nil)
	require.ErrorIs(t, err, storeErr)
	require.Equal(t, 1, store.calls)
	require.Equal(t, []any{"a", "b", "c"}, store.keys)

	attemptedValues = gatherKvCounters(t, registry, "gosoline_kvstore_attempted_reads_total")
	require.Equal(t, 3.0, attemptedValues[kvCounterKey{model: model, store: storeName}])
	readValues := gatherKvCounters(t, registry, "gosoline_kvstore_reads_total")
	require.Equal(t, map[kvCounterKey]float64{
		{model: model, store: storeName, hit: "true"}:  0.0,
		{model: model, store: storeName, hit: "false"}: 0.0,
	}, readValues)
}
