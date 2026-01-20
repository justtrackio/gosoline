package metric

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/justtrackio/gosoline/pkg/clock"
)

func TestWriter_ReplacesNaNWithCounterMetric(t *testing.T) {
	ch := &metricChannel{enabled: true, hasData: make(chan struct{}, 1)}
	clk := clock.NewFakeClockAt(time.Unix(1234567890, 0))
	w := NewWriterWithInterfaces(clk, ch)

	orig := &Datum{
		MetricName: "test_metric",
		Value:      math.NaN(),
		Unit:       UnitCount,
		Dimensions: map[string]string{"foo": "bar"},
		Kind:       KindCounter.Build(),
	}

	w.Write(context.Background(), Data{orig})

	ch.lck.Lock()
	written := ch.data
	ch.lck.Unlock()

	assert.Len(t, written, 1)
	m := written[0]
	assert.Equal(t, "metric_writer_nan_count", m.MetricName)
	assert.Equal(t, 1.0, m.Value)
	assert.Equal(t, UnitCount, m.Unit)
	assert.Equal(t, Dimensions{
		"metric_name": "test_metric",
	}, m.Dimensions)
	assert.False(t, m.Timestamp.IsZero())
}
