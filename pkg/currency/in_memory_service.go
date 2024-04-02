package currency

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
)

// NewInMemoryService creates a currency Service which never updates and reads all currency values in the constructor.
// It can be useful when you need to run a tool or similar and don't want to store the currency data somewhere or have
// a complicated module setup.
func NewInMemoryService(ctx context.Context, config cfg.Config, logger log.Logger) (Service, error) {
	logger = logger.WithChannel("in-memory-currency-service")

	store, err := kvstore.NewInMemoryKvStore[float64](ctx, config, logger, &kvstore.Settings{
		Ttl: time.Hour * 24 * 365, // data never expires or updates
		InMemorySettings: kvstore.InMemorySettings{
			MaxSize: 1_000_000_000,
		},
	})
	if err != nil {
		return nil, fmt.Errorf(": %w", err)
	}

	httpClient, err := http.ProvideHttpClient(ctx, config, logger, "currencyUpdater")
	if err != nil {
		return nil, fmt.Errorf("can not create http client: %w", err)
	}

	updater := NewUpdaterWithInterfaces(logger, store, httpClient, clock.Provider)

	err = updater.EnsureRecentExchangeRates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent exchange rates: %w", err)
	}

	err = updater.EnsureHistoricalExchangeRates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical exchange rates: %w", err)
	}

	return NewWithInterfaces(store, clock.Provider), nil
}
