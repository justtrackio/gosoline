package tracing

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Provider func(config cfg.Config, logger log.Logger) (Tracer, error)

func AddProvider(name string, provider Provider) {
	providers[name] = provider
}

var providers = map[string]Provider{}
