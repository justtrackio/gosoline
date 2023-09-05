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
	// api server factory which defines the http routes
	definer := func(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
		def := &apiserver.Definitions{}

		var err error
		var handler apiserver.HandlerWithInput

		// create a handler which handles input data
		if handler, err = NewTodoHandler(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not create trip handler: %w", err)
		}

		// bind this handler with input form the query params to the todo path with GET method
		def.GET("/todo", apiserver.CreateQueryHandler(handler))

		return def, nil
	}

	// runs an api server application based on the definitions
	application.RunApiServer(definer)
}
