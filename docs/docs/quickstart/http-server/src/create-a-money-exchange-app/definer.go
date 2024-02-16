// snippet-start: imports
package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

// snippet-end: imports

// snippet-start: api definer
func ApiDefiner(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
	// Create an empty Definitions object, called definitions.
	definitions := &httpserver.Definitions{}

	// Create a new euroHandler.
	euroHandler, err := NewEuroHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create euroHandler: %w", err)
	}

	// Create a new euroAtDateHandler.
	euroAtDateHandler, err := NewEuroAtDateHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create euroAtDateHandler: %w", err)
	}

	// Add two routes to definitions. Each route handles GET requests. Notice that each route uses one of the handlers you wrote in handlers.go.
	definitions.GET("/euro/:amount/:currency", httpserver.CreateHandler(euroHandler))
	definitions.GET("/euro-at-date/:amount/:currency/:date", httpserver.CreateHandler(euroAtDateHandler))

	// Return definitions.
	return definitions, nil
}

// snippet-end: api definer
