// snippet-start: imports
package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	"github.com/justtrackio/gosoline/pkg/log"
)

// snippet-end: imports

// snippet-start: main
func main() {
	definer := func(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
		// Instantiated a new Definitions object.
		def := &httpserver.Definitions{}

		var err error
		var handler crud.Handler

		// Created a new CRUD handler.
		if handler, err = NewTodoCrudHandler(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not create trip handler: %w", err)
		}

		// Add CRUD handlers to your definitions. This is a convenience method for adding handlers for Create, Read, Update, Delete, and List.
		if err := crud.AddCrudHandlers(config, logger, def, 0, "todo", handler); err != nil {
			return nil, fmt.Errorf("can not add crud handlers: %w", err)
		}

		return def, nil
	}

	// Run your server with the definitions.
	application.RunHttpDefaultServer(definer,
		application.WithConfigFile("config.dist.yml", "yml"),
	)
}

// snippet-end: main
