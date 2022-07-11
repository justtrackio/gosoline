package currency

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ProviderFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (Provider, error)

var providerFactories = map[string]ProviderFactory{}

func AddProviderFactory(providerType string, provider ProviderFactory) {
	providerFactories[providerType] = provider
}

func GetProviderFactory(providerType string) (ProviderFactory, bool) {
	provider, ok := providerFactories[providerType]
	if !ok {
		return nil, false
	}

	return provider, true
}
