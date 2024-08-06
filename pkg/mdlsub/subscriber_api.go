package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func CreateDefiner(callbacks map[string]stream.ConsumerCallbackFactory) httpserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
		d := &httpserver.Definitions{}

		for name, callback := range callbacks {
			handler, err := NewHandler(ctx, config, logger, callback)
			if err != nil {
				return nil, fmt.Errorf("could not define handler for %s: %w", name, err)
			}

			d.POST("/v0/subscription/"+name, httpserver.CreateJsonHandler(handler))
		}

		return d, nil
	}
}
