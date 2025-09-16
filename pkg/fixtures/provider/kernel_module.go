//go:build fixtures

package provider

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func definer(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
	d := &httpserver.Definitions{}

	handler, err := NewFixturesProviderHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create fixtures-provider handler: %w", err)
	}

	d.GET("/v0/fixtures/:store", httpserver.CreateHandler(handler))

	return d, nil
}

func NewFixturesProviderHttpServerModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	var enabled bool
	var err error
	if enabled, err = config.GetBool("httpserver.fixtures-provider.enabled", false); err != nil {
		return nil, fmt.Errorf("httpserver.fixtures-provider.enabled config key could not be read")
	}
	if !enabled {
		logger.Info(ctx, "fixtures provider httpserver is not enabled")

		return nil, nil
	}

	return httpserver.New("fixtures-provider", definer)(ctx, config, logger)
}
