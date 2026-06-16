package kafka_test

import (
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/kafka"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"
)

const writeGraceTime = 10 * time.Second

func TestMetricsHook_BrokerConnect_ConsistentLabels(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	registry := prometheus.NewRegistry()
	writer := metric.NewPrometheusWriterWithInterfaces(logger, registry, "test", 1000, writeGraceTime)

	producerHook := kafka.NewMetricsHook(writer, kafka.DimensionProducer, "my-producer")
	consumerHook := kafka.NewMetricsHook(writer, kafka.DimensionConsumer, "my-consumer")

	meta := kgo.BrokerMetadata{Host: "localhost", Port: 9092}

	// Both hooks write the same metric name with the same label names (ClientType, Client, Broker).
	producerHook.OnBrokerConnect(meta, 0, nil, nil)
	consumerHook.OnBrokerConnect(meta, 0, nil, nil)

	count, err := testutil.GatherAndCount(registry, "test_BrokerConnects")
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "expected 2 time series for BrokerConnects (one per client type)")
}

func TestMetricsHook_BrokerConnect_VariableLabelsCauseConflict(t *testing.T) {
	registry := prometheus.NewRegistry()

	// Simulate the old broken behavior: registering the same metric name with different label names.
	counter1 := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "test",
		Name:      "OldStyleMetric",
		Help:      "unit: Count",
	}, []string{"Producer", "Broker"})

	counter2 := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "test",
		Name:      "OldStyleMetric",
		Help:      "unit: Count",
	}, []string{"Consumer", "Broker"})

	// First registration succeeds.
	err := registry.Register(counter1)
	assert.NoError(t, err)

	// Second registration with different labels fails — this is the bug we fixed.
	err = registry.Register(counter2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "different label names")
}
