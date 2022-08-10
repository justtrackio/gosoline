package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type SubscriberApiSettings struct {
	Enabled bool `cfg:"enabled" default:"true"`
}

func CreateDefiner(callbacks map[string]stream.ConsumerCallbackFactory) apiserver.Definer {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
		d := &apiserver.Definitions{}

		for name, callback := range callbacks {
			handler, err := NewHandler(ctx, config, logger, callback)
			if err != nil {
				return nil, fmt.Errorf("could not define handler for %s: %w", name, err)
			}

			d.POST("/v0/subscription/"+name, apiserver.CreateJsonHandler(handler))
			// TODO add batch handler
		}

		return d, nil
	}
}
