// Package appctx provides a [context.Context] based mechanism of sharing certain "global" resources
// across different parts of an application.
// When building applications with the [github.com/justtrackio/gosoline/pkg/application] package
// a Container is automatically injected into the application's context.
package appctx

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ContextValueFactory[T any] func() (key any, provider func(ctx context.Context, config cfg.Config, logger log.Logger) (T, error))
