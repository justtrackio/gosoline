//go:build !fixtures

package provider

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewFixturesProviderHttpServerModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	return nil, nil
}
