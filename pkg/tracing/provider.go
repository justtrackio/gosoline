package tracing

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type Provider func(config cfg.Config, logger mon.Logger) Tracer

func AddProvider(name string, provider Provider) {
	providers[name] = provider
}

var providers = map[string]Provider{
	"xray": NewAwsTracer,
}
