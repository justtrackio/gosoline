package producer

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

type producerWriterFunc func(context.Context, ...*kgo.Record) kgo.ProduceResults

func (f producerWriterFunc) ProduceSync(ctx context.Context, records ...*kgo.Record) kgo.ProduceResults {
	return f(ctx, records...)
}

func TestProducerProduceSyncExportsHistogramsWithInjectedWriter(t *testing.T) {
	registry := prometheus.NewRegistry()
	metricWriter := metric.NewPrometheusWriterWithInterfaces(log.NewLogger(), registry, 1000, time.Second)
	testClock := clock.NewFakeClock()
	record := &kgo.Record{Value: []byte("test")}
	writer := producerWriterFunc(func(context.Context, ...*kgo.Record) kgo.ProduceResults {
		testClock.Advance(time.Second)

		return kgo.ProduceResults{{Record: record}}
	})

	p := NewProducerWithInterfaces(writer, metricWriter, "test-producer", "test-topic").(*producer)
	p.clock = testClock
	assert.NoError(t, p.ProduceSync(context.Background(), record))

	families, err := registry.Gather()
	require.NoError(t, err)

	familiesByName := make(map[string]*dto.MetricFamily, len(families))
	for _, family := range families {
		familiesByName[family.GetName()] = family
	}

	assertHistogramFamily(t, familiesByName, "gosoline_kafka_producer_batch_records", prometheus.DefBuckets)
	assertHistogramFamily(t, familiesByName, "gosoline_kafka_producer_produce_duration_seconds", []float64{
		0.005,
		0.01,
		0.025,
		0.05,
		0.075,
		0.1,
		0.25,
		0.5,
		0.75,
		1,
		2.5,
		5,
		7.5,
		10,
	})

	counterFamily, ok := familiesByName["gosoline_kafka_producer_sent_messages_total"]
	if assert.True(t, ok, "missing Prometheus counter family") && assert.Equal(t, dto.MetricType_COUNTER, counterFamily.GetType()) && assert.Len(t, counterFamily.GetMetric(), 1) {
		assert.Equal(t, float64(1), counterFamily.GetMetric()[0].GetCounter().GetValue())
	}
}

func assertHistogramFamily(t *testing.T, families map[string]*dto.MetricFamily, name string, expectedBuckets []float64) {
	t.Helper()

	family, ok := families[name]
	if !assert.True(t, ok, "missing Prometheus family %s", name) {
		return
	}

	if !assert.Equal(t, dto.MetricType_HISTOGRAM, family.GetType()) || !assert.Len(t, family.GetMetric(), 1) {
		return
	}

	histogram := family.GetMetric()[0].GetHistogram()
	assert.Equal(t, uint64(1), histogram.GetSampleCount())
	assert.Equal(t, float64(1), histogram.GetSampleSum())
	require.Len(t, histogram.GetBucket(), len(expectedBuckets))
	for i, bound := range expectedBuckets {
		assert.Equal(t, bound, histogram.GetBucket()[i].GetUpperBound())
	}
	// The DTO represents the +Inf bucket implicitly through SampleCount.
	assert.Equal(t, uint64(1), histogram.GetBucket()[len(expectedBuckets)-1].GetCumulativeCount())
}
