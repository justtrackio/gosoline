package currency

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	ExchangeRateRefresh           = 8 * time.Hour
	ExchangeRateDateKey           = "currency_exchange_last_refresh"
	HistoricalExchangeRateDateKey = "currency_exchange_historical_last_refresh"
)

// RatesAmount
// We expect to have around 200 currencies in total.
// It's okay to exceed the limit, but it's nice to avoid unnecessary reallocations.
const RatesAmount = 200

type Settings struct {
	StartDate time.Time `cfg:"start_date"`
}

type ProvidersConfig map[string]ProviderSettings

//go:generate go run github.com/vektra/mockery/v2 --name UpdaterService
type UpdaterService interface {
	EnsureRecentExchangeRates(ctx context.Context) error
	EnsureHistoricalExchangeRates(ctx context.Context) error
}

type updaterService struct {
	logger    log.Logger
	http      http.Client
	store     kvstore.KvStore[float64]
	clock     clock.Clock
	providers []Provider
	settings  *Settings
}

func NewUpdater(ctx context.Context, config cfg.Config, logger log.Logger) (UpdaterService, error) {
	logger = logger.WithChannel("currency_updater_service")

	store, err := kvstore.ProvideConfigurableKvStore[float64](ctx, config, logger, kvStoreName)
	if err != nil {
		return nil, fmt.Errorf("can not create kvStore: %w", err)
	}

	httpClient, err := http.ProvideHttpClient(ctx, config, logger, "currencyUpdater")
	if err != nil {
		return nil, fmt.Errorf("can not create http client: %w", err)
	}

	settings := &Settings{}
	if err := config.UnmarshalKey("currency_service", settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal currency updater settings: %w", err)
	}

	if settings.StartDate.IsZero() {
		settings.StartDate = clock.Provider.Now().AddDate(0, -1, 0)
	}

	providers, err := initProviders(ctx, config, logger, httpClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize providers: %w", err)
	}

	return NewUpdaterWithInterfaces(logger, store, httpClient, clock.Provider, settings, providers), nil
}

func NewUpdaterWithInterfaces(logger log.Logger, store kvstore.KvStore[float64], httpClient http.Client, clock clock.Clock, settings *Settings, providers []Provider) UpdaterService {
	return &updaterService{
		logger:    logger,
		store:     store,
		http:      httpClient,
		clock:     clock,
		providers: providers,
		settings:  settings,
	}
}

func (s *updaterService) EnsureRecentExchangeRates(ctx context.Context) error {
	if !s.needsRefresh(ctx) {
		return nil
	}

	s.logger.Info(ctx, "requesting exchange rates")
	rates, err := s.getCurrencyRates(ctx)
	if err != nil {
		return fmt.Errorf("error getting currency exchange rates: %w", err)
	}

	now := s.clock.Now()
	for _, rate := range rates {
		err := s.store.Put(ctx, rate.Currency, rate.Rate)
		if err != nil {
			return fmt.Errorf("error setting exchange rate: %w", err)
		}

		s.logger.Info(ctx, "currency: %s, rate: %f", rate.Currency, rate.Rate)

		historicalRateKey := historicalRateKey(now, rate.Currency)
		err = s.store.Put(ctx, historicalRateKey, rate.Rate)
		if err != nil {
			return fmt.Errorf("error setting historical exchange rate, key: %s %w", historicalRateKey, err)
		}
	}

	newTime := float64(s.clock.Now().Unix())

	err = s.store.Put(ctx, ExchangeRateDateKey, newTime)
	if err != nil {
		return fmt.Errorf("error setting refresh date %w", err)
	}

	s.logger.Info(ctx, "new exchange rates are set")

	return nil
}

func (s *updaterService) needsRefresh(ctx context.Context) bool {
	var dateUnix float64
	exists, err := s.store.Get(ctx, ExchangeRateDateKey, &dateUnix)
	if err != nil {
		s.logger.Info(ctx, "error fetching date")

		return true
	}

	if !exists {
		s.logger.Info(ctx, "date doesn't exist")

		return true
	}

	comparisonDate := s.clock.Now().Add(-ExchangeRateRefresh)

	date := time.Unix(int64(dateUnix), 0)

	if date.Before(comparisonDate) {
		s.logger.Info(ctx, "comparison date %s was more than 8 hours ago", date.Format(time.DateTime))

		return true
	}

	return false
}

func (s *updaterService) getCurrencyRates(ctx context.Context) ([]Rate, error) {
	allRates := make([]Rate, 0, RatesAmount)
	rateMap := make(funk.Set[string])

	for _, provider := range s.providers {
		rates, err := provider.FetchLatestRates(ctx)
		if err != nil {
			return nil, fmt.Errorf("error getting rates from %s: %w", provider.Name(), err)
		}

		for _, rate := range rates {
			if _, exists := rateMap[rate.Currency]; !exists {
				allRates = append(allRates, rate)
				rateMap.Add(rate.Currency)
			}
		}
	}

	return allRates, nil
}

func (s *updaterService) EnsureHistoricalExchangeRates(ctx context.Context) error {
	updateFromDate := s.updateFromDate(ctx)
	if updateFromDate == nil {
		return nil
	}

	s.logger.Info(ctx, "requesting historical exchange rates from %s", updateFromDate.Format(time.DateOnly))

	datesToUpdate := generateDateRange(*updateFromDate, s.clock.Now())

	providerResults := s.fetchHistoricalProviderResults(ctx, datesToUpdate)

	dayCurrencyRates := mergeProviderResults(datesToUpdate, providerResults)

	fillHistoricalGaps(dayCurrencyRates)

	// Preparing data for storing in kvstore
	keyValues := buildHistoricalKeyValues(datesToUpdate, dayCurrencyRates)

	s.logger.Info(ctx, "updating %d historical exchange rates", len(keyValues))

	err := s.store.PutBatch(ctx, keyValues)
	if err != nil {
		return fmt.Errorf("error setting historical exchange rates: %w", err)
	}

	newTime := float64(s.clock.Now().Unix())
	err = s.store.Put(ctx, HistoricalExchangeRateDateKey, newTime)
	if err != nil {
		return fmt.Errorf("error setting historical refresh date %w", err)
	}

	s.logger.Info(ctx, "stored %d day-currency combinations of historical exchange rates", len(keyValues))

	return nil
}

func (s *updaterService) fetchHistoricalProviderResults(ctx context.Context, datesToUpdate []time.Time) []map[time.Time][]Rate {
	providerResults := make([]map[time.Time][]Rate, len(s.providers))
	for i, provider := range s.providers {
		result, err := provider.FetchHistoricalRates(ctx, datesToUpdate)
		if err != nil {
			s.logger.Error(ctx, "error fetching historical rates from provider %s: %w", provider.Name(), err)

			continue
		}

		providerResults[i] = result
	}

	return providerResults
}

// mergeProviderResults gathers all results, provider by provider, day by day.
// Higher priority providers are preferred over lower priority ones.
func mergeProviderResults(datesToUpdate []time.Time, providerResults []map[time.Time][]Rate) map[time.Time]map[string]float64 {
	dayCurrencyRates := make(map[time.Time]map[string]float64)
	for _, date := range datesToUpdate {
		dayCurrencyRates[date] = make(map[string]float64)

		for _, providerResult := range providerResults {
			if dayRates, ok := providerResult[date]; ok {
				for _, rate := range dayRates {
					if _, exists := dayCurrencyRates[date][rate.Currency]; !exists {
						dayCurrencyRates[date][rate.Currency] = rate.Rate
					}
				}
			}
		}
	}

	return dayCurrencyRates
}

func buildHistoricalKeyValues(datesToUpdate []time.Time, dayCurrencyRates map[time.Time]map[string]float64) map[string]float64 {
	keyValues := make(map[string]float64)
	for _, d := range datesToUpdate {
		for currency, rate := range dayCurrencyRates[d] {
			key := historicalRateKey(d, currency)
			keyValues[key] = rate
		}
	}

	return keyValues
}

func initProviders(ctx context.Context, config cfg.Config, logger log.Logger, httpClient http.Client) ([]Provider, error) {
	var providers []Provider
	providersConfig := make(ProvidersConfig)

	for name := range providerRegistry {
		providerSettings := ProviderSettings{}
		if err := config.UnmarshalKey("currency_service.providers."+name, &providerSettings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal provider settings for %s: %w", name, err)
		}

		providersConfig[name] = providerSettings
	}

	for name, providerSettings := range providersConfig {
		option, ok := providerRegistry[name]
		if !ok {
			logger.Warn(ctx, "provider %s not found in registry, skipping", name)

			continue
		}

		provider := option(ctx, logger, httpClient, providerSettings)
		if provider != nil {
			logger.Info(ctx, "registering currency provider: %s", provider.Name())
			providers = append(providers, provider)
		} else {
			logger.Info(ctx, "currency provider %s is disabled ", name)
		}
	}

	// priority is used to define which rate will be preferred when multiple providers return rates for the same currency
	// lower numbers are preferred over higher numbers (ascending order)
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].GetPriority() < providers[j].GetPriority()
	})

	return providers, nil
}

func generateDateRange(start, end time.Time) []time.Time {
	var dates []time.Time

	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}

	return dates
}

func fillHistoricalGaps(dayCurrencyRates map[time.Time]map[string]float64) {
	dates := funk.Keys(dayCurrencyRates)

	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

	for i := 0; i < len(dates)-1; i++ {
		currentDate := dates[i]
		nextDay := dates[i+1]

		for currency, rate := range dayCurrencyRates[currentDate] {
			if _, ok := dayCurrencyRates[nextDay][currency]; !ok {
				dayCurrencyRates[nextDay][currency] = rate
			}
		}
	}
}

func (s *updaterService) updateFromDate(ctx context.Context) *time.Time {
	var dateUnix float64

	exists, err := s.store.Get(ctx, HistoricalExchangeRateDateKey, &dateUnix)
	if err != nil {
		s.logger.Info(ctx, "updateFromDate error fetching date, using start date")

		return &s.settings.StartDate
	}

	if !exists {
		s.logger.Info(ctx, "updateFromDate date doesn't exist")

		return &s.settings.StartDate
	}

	comparisonDate := s.clock.Now().Add(-24 * time.Hour)

	date := time.Unix(int64(dateUnix), 0)

	if date.Before(comparisonDate) {
		s.logger.Info(ctx, "updateFromDate comparison date was more than threshold")

		return &date
	}

	return nil
}

func historicalRateKey(date time.Time, currency string) string {
	return date.Format(time.DateOnly) + "-" + currency
}
