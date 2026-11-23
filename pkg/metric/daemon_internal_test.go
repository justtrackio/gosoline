package metric

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudWatchAggregationDefault(t *testing.T) {
	tests := []struct {
		name     string
		settings map[string]any
		want     []float64
	}{
		{name: "absent", want: []float64{5}},
		{name: "disabled", settings: map[string]any{"aggregate": false}, want: []float64{2, 3}},
		{name: "enabled", settings: map[string]any{"aggregate": true}, want: []float64{5}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := appctx.WithContainer(t.Context())
			sink := &recordingCloudWatchWriter{}
			_, err := appctx.Provide(ctx, cwWriterCtxKey("default"), func() (Writer, error) {
				return sink, nil
			})
			require.NoError(t, err)

			config := cfg.New(map[string]any{
				"metric": map[string]any{
					"enabled": true,
					"writers": []string{WriterTypeCloudwatch},
					"writer_settings": map[string]any{
						WriterTypeCloudwatch: tt.settings,
					},
				},
			})
			module, err := NewDaemonModule(ctx, config, log.NewCliLogger())
			require.NoError(t, err)
			require.IsType(t, (*Daemon)(nil), module)
			daemon := module.(*Daemon)
			t.Cleanup(daemon.ticker.Stop)

			ts := time.Unix(0, 0).UTC()
			data := Data{
				{MetricName: "requests", Priority: PriorityHigh, Timestamp: ts, Unit: UnitCount, Value: 2},
				{MetricName: "requests", Priority: PriorityHigh, Timestamp: ts, Unit: UnitCount, Value: 3},
			}
			daemon.rawFanout(ctx, data)
			daemon.appendBatch(ctx, data)
			daemon.publish(ctx)

			require.Equal(t, tt.want, sink.values)
		})
	}
}

type recordingCloudWatchWriter struct {
	values []float64
}

func (*recordingCloudWatchWriter) GetPriority() int {
	return PriorityHigh
}

func (w *recordingCloudWatchWriter) Write(_ context.Context, data Data) {
	for _, datum := range data {
		w.values = append(w.values, datum.Value)
	}
}

func (w *recordingCloudWatchWriter) WriteOne(ctx context.Context, datum *Datum) {
	w.Write(ctx, Data{datum})
}

func TestDaemonThrottleErrorExpires(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	daemon := newThrottleTestDaemon(fakeClock)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)

	assert.True(t, daemon.throttleError(ctx, "invalid datum"))
	assert.False(t, daemon.throttleError(ctx, "invalid datum"))

	fakeClock.BlockUntil(1)
	fakeClock.Advance(time.Minute)

	require.Eventually(t, func() bool {
		return daemon.throttleError(ctx, "invalid datum")
	}, time.Second, time.Millisecond)
}

func TestDaemonThrottleErrorReleasesOnContextCancellation(t *testing.T) {
	fakeClock := clock.NewFakeClock()
	daemon := newThrottleTestDaemon(fakeClock)
	ctx, cancel := context.WithCancel(t.Context())

	assert.True(t, daemon.throttleError(ctx, "invalid datum"))
	fakeClock.BlockUntil(1)
	cancel()

	nextCtx, nextCancel := context.WithCancel(t.Context())
	t.Cleanup(nextCancel)
	require.Eventually(t, func() bool {
		return daemon.throttleError(nextCtx, "invalid datum")
	}, time.Second, time.Millisecond)
}

func newThrottleTestDaemon(clk clock.Clock) *Daemon {
	return newMetricDaemonWithInterfaces(log.NewCliLogger(), nil, nil, nil, &Settings{
		Interval: time.Minute,
	}, clk)
}
