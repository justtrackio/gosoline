package mdlsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/stream"
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
		}

		return d, nil
	}
}
