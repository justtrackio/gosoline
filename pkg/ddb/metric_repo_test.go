package ddb

import (
	"context"
	"errors"
	"testing"
	"time"

	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

type metricTestRepository struct {
	Repository
	modelId mdl.ModelId
	err     error
}

func (r metricTestRepository) GetModelId() mdl.ModelId {
	return r.modelId
}

func (r metricTestRepository) PutItem(context.Context, PutItemBuilder, any) (*PutItemResult, error) {
	return nil, r.err
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
		if family.GetName() == "gosoline_ddb_operations_total" {
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

func TestMetricRepository_PutItemFailureEmitsCounterAndSeedsOutcomes(t *testing.T) {
	modelId := mdl.ModelId{Name: "users", DomainPattern: "storage"}
	modelIdString := modelId.String()

	registry := prometheus.NewRegistry()
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	exporter := metric.NewPrometheusWriterWithInterfaces(logger, registry, 1000, time.Second)
	writer := namespacedMetricWriter{Writer: exporter, namespace: metricNamespace}
	defaults := getDefaultMetrics(modelId)
	for _, datum := range defaults {
		if datum.MetricName == metricNameOperations {
			require.Equal(t, metric.PriorityHigh, datum.Priority)
			require.Equal(t, metric.UnitCount, datum.Unit)
			require.Equal(t, metric.KindCounter.Build(), datum.Kind)
		}
	}
	writer.Write(t.Context(), defaults)

	values := gatherOperationCounters(t, registry)
	require.Len(t, values, 2)
	require.Equal(t, 0.0, values[operationCounterKey{operation: OpSave, modelId: modelIdString}])
	require.Equal(t, 0.0, values[operationCounterKey{operation: OpSave, modelId: modelIdString, errorType: errorTypeOperationFailed}])

	operationErr := errors.New("DynamoDB operation failed")
	repo := metricTestRepository{modelId: modelId, err: operationErr}
	wrapped := metricRepository{Repository: repo, metric: writer}
	_, err := wrapped.PutItem(t.Context(), nil, nil)
	require.ErrorIs(t, err, operationErr)

	values = gatherOperationCounters(t, registry)
	require.Equal(t, 1.0, values[operationCounterKey{operation: OpSave, modelId: modelIdString, errorType: errorTypeOperationFailed}])
	require.Equal(t, 0.0, values[operationCounterKey{operation: OpSave, modelId: modelIdString}])
}
