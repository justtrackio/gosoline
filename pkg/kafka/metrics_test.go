package kafka

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricPairReturnsOnlyPartitionSeries(t *testing.T) {
	kind := metric.KindHistogram.Build()
	data := MetricPair(MetricSpec{
		ClientType: ClientTypeConsumer,
		ClientName: "reader",
		Namespace:  "kafka.consumer",
		Name:       "poll.duration",
		Topic:      "orders",
		Partition:  3,
		Value:      1,
		Unit:       metric.UnitMillisecondsAverage,
		Kind:       kind,
	})

	require.Len(t, data, 1)
	assert.Equal(t, metric.Dimensions{
		DimensionClientType: ClientTypeConsumer,
		DimensionClient:     "reader",
		DimensionTopic:      "orders",
		DimensionPartition:  "3",
	}, data[0].Dimensions)
	assert.Equal(t, kind, data[0].Kind)
}
