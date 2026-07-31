package metric

import (
	"context"
	"errors"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"go.opentelemetry.io/otel/sdk/metric"
)

type (
	metricShutdownKey struct{}
)

var _ kernel.ShutdownHandler = &ShutdownHandler{}

// ProvideShutdownHandler returns a ShutdownHandler that retrieves the metric provider's
// shutdown function from the appctx container and invokes it.
func ProvideShutdownHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*ShutdownHandler, error) {
	return appctx.Provide(ctx, metricShutdownKey{}, func() (*ShutdownHandler, error) {
		return &ShutdownHandler{}, nil
	})
}

type ShutdownHandler struct {
	providers []*metric.MeterProvider
}

func (h *ShutdownHandler) AddProvider(provider *metric.MeterProvider) {
	h.providers = append(h.providers, provider)
}

// Shutdown retrieves the registered metric provider shutdown function from the appctx
// container. If no provider was registered, it is a no-op.
func (h *ShutdownHandler) Shutdown(ctx context.Context) error {
	var err error

	for _, p := range h.providers {
		if sErr := p.Shutdown(ctx); sErr != nil {
			err = errors.Join(err, sErr)
		}
	}

	return err
}
