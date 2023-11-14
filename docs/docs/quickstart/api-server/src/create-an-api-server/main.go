// snippet-start: imports
package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)
// snippet-end: imports

// snippet-start: main
func main() {
	// Initialize an API server factory that defines your HTTP route
	definer := func(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
		// Initialize a reference to apiserver.Definitions, which you use to create a GET route
		def := &apiserver.Definitions{}

		// Instantiate two new variables for handling errors and request input data
		var err error
		var handler apiserver.HandlerWithInput

		// Create a handler (`NewTodoHandler`) that handles the request input data. 
		// If there is an error, return an error message. 
		// If there is no error, assign this `NewTodoHandler` to the `handler` variable from the last step.
		if handler, err = NewTodoHandler(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not create trip handler: %w", err)
		}

		// Create a GET route for the endpoint /todo, using the handler
		def.GET("/todo", apiserver.CreateQueryHandler(handler))

		// Return the response from handler
		return def, nil
	}

	// Run an API server application based on the logic from the previous steps
	application.RunApiServer(definer)
}
// snippet-end: main
