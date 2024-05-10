package metric_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/stretchr/testify/assert"
)

// ensure the metric daemon implements the typed and staged module interfaces
var _ interface {
	kernel.Module
	kernel.TypedModule
	kernel.StagedModule
} = &metric.Daemon{}

type slowWriter struct {
	ctx context.Context
}

func (s slowWriter) GetPriority() int {
	return metric.PriorityHigh
}

func (s slowWriter) Write(metric.Data) {
	select {
	case <-s.ctx.Done():
	case <-time.After(time.Second):
	}
}

func (s slowWriter) WriteOne(data *metric.Datum) {
	s.Write(metric.Data{data})
}

func TestWriteLotsOfBadMetrics(t *testing.T) {
	metric.RegisterWriterFactory("test", func(ctx context.Context, config cfg.Config, logger log.Logger) (metric.Writer, error) {
		return slowWriter{
			ctx: ctx,
		}, nil
	})

	ctx, cancel := context.WithCancel(appctx.WithContainer(t.Context()))

	config := cfg.New(map[string]any{
		"env":         "test",
		"app_project": "justtrack",
		"app_family":  "gosoline",
		"app_group":   "gosoline",
		"app_name":    "metric_daemon_test",
		"metric": map[string]any{
			"enabled":  true,
			"interval": "1s",
			"writer":   "test",
		},
	})
	logger := log.NewCliLogger()

	daemon, err := metric.NewDaemonModule(ctx, config, logger)
	assert.NoError(t, err)

	writer := metric.NewWriter(&metric.Datum{
		MetricName: "myMetricName",
	})

	cfn := coffin.New()
	cfn.GoWithContext(ctx, daemon.Run)
	cfn.GoWithContext(ctx, func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Millisecond * 10):
			}

			writer.WriteOne(&metric.Datum{
				MetricName: "myMetricName",
			})
		}
	})
	for i := 0; i < 10; i++ {
		cfn.GoWithContext(ctx, func(ctx context.Context) error {
			for {
				select {
				case <-ctx.Done():
					return nil
				default:
				}

				writer.WriteOne(&metric.Datum{
					Priority:   metric.PriorityHigh,
					MetricName: "myOtherMetricName",
					Unit:       metric.UnitCount,
					Value:      1,
				})
			}
		})
	}
	cfn.Go(func() error {
		time.Sleep(10 * time.Second)
		cancel()

		return nil
	})

	err = cfn.Wait()
	assert.NoError(t, err)
}
