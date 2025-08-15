package currency

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ProviderSettings struct {
	ApiKey string `cfg:"api_key"`
	// Providers with lower values will be used first. The order of the providers with the same priority is non-deterministic.
	Priority uint `cfg:"priority"`
	Enabled  bool `cfg:"enabled"`
}

//go:generate go run github.com/vektra/mockery/v2 --name Provider
type Provider interface {
	Name() string
	GetPriority() int
	FetchLatestRates(ctx context.Context) ([]Rate, error)
	FetchHistoricalRates(ctx context.Context, dates []time.Time) (map[time.Time][]Rate, error)
}

var providerRegistry = map[string]ProviderFactory{
	ECBProviderName:                  newECBProvider,
	OpenExchangeRatesApiProviderName: newOpenExchangeRatesApiProvider,
}

type ProviderFactory func(ctx context.Context, logger log.Logger, http http.Client, settings ProviderSettings) Provider

// RegisterProvider registers a new currency provider with the given name and factory.
// If a provider with the same name already exists, it will be overwritten.
// Providers should be registered before currency module is initialized (e.g. in init function).
func RegisterProvider(name string, option ProviderFactory) {
	if _, exists := providerRegistry[name]; exists {
		panic("a currency provider with the name " + name + " already exists")
	}

	providerRegistry[name] = option
}
