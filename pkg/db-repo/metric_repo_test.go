package db_repo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type metricTestRepository struct {
	Repository
	modelId string
	err     error
}

func (r metricTestRepository) GetModelId() string {
	return r.modelId
}

func (r metricTestRepository) Create(context.Context, ModelBased) error {
	return r.err
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

type operationCounterKey struct {
	operation string
	modelId   string
	errorType string
}

func gatherOperationCounters(t *testing.T, registry *prometheus.Registry) map[operationCounterKey]float64 {
	t.Helper()

	families, err := registry.Gather()
	require.NoError(t, err)

	var counterFamily *dto.MetricFamily
	for _, family := range families {
		if family.GetName() == "gosoline_db_repo_operations_total" {
			counterFamily = family

			break
		}
	}
	require.NotNil(t, counterFamily)
	require.Equal(t, dto.MetricType_COUNTER, counterFamily.GetType())

	values := make(map[operationCounterKey]float64, len(counterFamily.GetMetric()))
	for _, sample := range counterFamily.GetMetric() {
		labels := make(map[string]string, len(sample.GetLabel()))
		for _, label := range sample.GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}

		values[operationCounterKey{
			operation: labels["operation_name"],
			modelId:   labels["model_id"],
			errorType: labels["error_type"],
		}] = sample.GetCounter().GetValue()
	}

	return values
}

func TestMetricRepository_CreateFailureEmitsCounterAndSeedsOutcomes(t *testing.T) {
	const modelId = "test-model"

	registry := prometheus.NewRegistry()
	logger := mocks.NewLoggerMock(mocks.WithMockAll, mocks.WithTestingT(t))
	exporter := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, time.Second)
	writer := namespacedMetricWriter{
		Writer:    exporter,
		namespace: metricNamespace,
	}
	defaults := getDefaultRepositoryMetrics(modelId)
	for _, datum := range defaults {
		if datum.MetricName == metricNameDbOperations {
			require.Equal(t, metric.PriorityHigh, datum.Priority)
			require.Equal(t, metric.UnitCount, datum.Unit)
			require.Equal(t, metric.KindCounter.Build(), datum.Kind)
		}
	}
	writer.Write(t.Context(), metric.Data(defaults))

	values := gatherOperationCounters(t, registry)
	require.Len(t, values, len(operations)*2)
	for _, operation := range operations {
		require.Equal(t, 0.0, values[operationCounterKey{
			operation: operation,
			modelId:   modelId,
		}])
		require.Equal(t, 0.0, values[operationCounterKey{
			operation: operation,
			modelId:   modelId,
			errorType: errorTypeOperationFailed,
		}])
	}

	operationErr := errors.New("database operation failed")
	repo := metricTestRepository{
		modelId: modelId,
		err:     operationErr,
	}
	wrapped := metricRepository{
		Repository: repo,
		output:     writer,
	}
	require.ErrorIs(t, wrapped.Create(t.Context(), nil), operationErr)

	values = gatherOperationCounters(t, registry)
	require.Equal(t, 1.0, values[operationCounterKey{
		operation: Create,
		modelId:   modelId,
		errorType: errorTypeOperationFailed,
	}])
	require.Equal(t, 0.0, values[operationCounterKey{
		operation: Create,
		modelId:   modelId,
	}])
}
