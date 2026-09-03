package db

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishConnectionMetricsStopsAfterContextCancellation(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	tickerStopped := make(chan struct{})
	batches := make(chan metric.Data, 3)
	conn := sqlx.NewDb(&sql.DB{}, "")
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	publishConnectionMetricsWithInterfaces(ctx, conn, tickerObservingClock{
		Clock:         fakeClock,
		tickerStopped: tickerStopped,
	}, connectionMetricWriter{batches: batches})

	assertConnectionMetricBatch(t, receiveConnectionMetricBatch(t, batches))

	fakeClock.BlockUntilTickers(1)
	fakeClock.Advance(time.Minute)
	assertConnectionMetricBatch(t, receiveConnectionMetricBatch(t, batches))

	cancel()
	select {
	case <-tickerStopped:
	case <-time.After(time.Second):
		t.Fatal("connection metric publisher did not stop after context cancellation")
	}

	fakeClock.Advance(time.Minute)
	select {
	case batch := <-batches:
		t.Fatalf("unexpected metric batch after context cancellation: %#v", batch)
	default:
	}
}

func assertConnectionMetricBatch(t *testing.T, batch metric.Data) {
	t.Helper()
	require.Len(t, batch, 2)
	assert.Equal(t, metricNameDbConnectionCount, batch[0].MetricName)
	assert.Equal(t, connectionStateUsed, batch[0].Dimensions[dimensionConnectionState])
	assert.Equal(t, metricNameDbConnectionCount, batch[1].MetricName)
	assert.Equal(t, connectionStateIdle, batch[1].Dimensions[dimensionConnectionState])
}

func receiveConnectionMetricBatch(t *testing.T, batches <-chan metric.Data) metric.Data {
	t.Helper()

	select {
	case batch := <-batches:
		return batch
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for connection metric batch")
		return nil
	}
}

type connectionMetricWriter struct {
	batches chan<- metric.Data
}

func (w connectionMetricWriter) GetPriority() int {
	return metric.PriorityLow
}

func (w connectionMetricWriter) Write(_ context.Context, batch metric.Data) {
	w.batches <- batch
}

func (w connectionMetricWriter) WriteOne(ctx context.Context, datum *metric.Datum) {
	w.Write(ctx, metric.Data{datum})
}

type tickerObservingClock struct {
	clock.Clock

	tickerStopped chan struct{}
}

func (c tickerObservingClock) NewTicker(d time.Duration) clock.Ticker {
	return &tickerObservingTicker{
		Ticker:  c.Clock.NewTicker(d),
		stopped: c.tickerStopped,
	}
}

type tickerObservingTicker struct {
	clock.Ticker

	stopped chan struct{}
	once    sync.Once
}

func (t *tickerObservingTicker) Stop() {
	t.Ticker.Stop()
	t.once.Do(func() {
		close(t.stopped)
	})
}
