//go:build integration && fixtures

package apitest

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

var ApiDefiner apiserver.Definer = func(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
	return nil, nil
}
