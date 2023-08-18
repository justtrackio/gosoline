package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	definer := func(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
		def := &apiserver.Definitions{}

		var err error
		var handler apiserver.HandlerWithInput

		if handler, err = NewTodoHandler(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not create trip handler: %w", err)
		}

		def.GET("/todo", apiserver.CreateQueryHandler(handler))

		return def, nil
	}

	application.RunApiServer(definer)
}
