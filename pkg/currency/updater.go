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

const (
	ExchangeRateRefresh           = 8 * time.Hour
	ExchangeRateDateKey           = "currency_exchange_last_refresh"
	HistoricalExchangeRateDateKey = "currency_exchange_historical_last_refresh"
	YMDLayout                     = "2006-01-02"
)

type UpdaterConfig struct {
	Provider string `cfg:"provider" default:"ecb"`
}

//go:generate mockery --name UpdaterService
type UpdaterService interface {
	EnsureRecentExchangeRates(ctx context.Context) error
	EnsureHistoricalExchangeRates(ctx context.Context) error
}

type updaterService struct {
	logger   log.Logger
	clock    clock.Clock
	http     http.Client
	provider Provider
	store    kvstore.KvStore
}

func NewUpdater(ctx context.Context, config cfg.Config, logger log.Logger) (UpdaterService, error) {
	logger = logger.WithChannel("currency_updater_service")

	clk := clock.Provider

	updaterConfig := UpdaterConfig{}
	config.UnmarshalKey("currency.updater", &updaterConfig)

	if updaterConfig.Provider == "" {
		return nil, fmt.Errorf("missing provider configuration currency.updater.provider")
	}

	providerFn, ok := GetProviderFactory(updaterConfig.Provider)
	if !ok {
		return nil, fmt.Errorf("undefined provider %s", updaterConfig.Provider)
	}

	provider, err := providerFn(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create provider: %w", err)
	}

	store, err := kvstore.ProvideConfigurableKvStore(ctx, config, logger, kvStoreName)
	if err != nil {
		return nil, fmt.Errorf("can not create kvStore: %w", err)
	}

	return NewUpdaterWithInterfaces(logger, clk, provider, store), nil
}

func NewUpdaterWithInterfaces(logger log.Logger, clock clock.Clock, provider Provider, store kvstore.KvStore) UpdaterService {
	return &updaterService{
		logger:   logger,
		clock:    clock,
		provider: provider,
		store:    store,
	}
}

func (s *updaterService) EnsureRecentExchangeRates(ctx context.Context) error {
	if !s.needsRefresh(ctx) {
		return nil
	}

	s.logger.Info("requesting exchange rates")
	rates, err := s.provider.FetchCurrentRates(ctx)
	if err != nil {
		return fmt.Errorf("error getting currency exchange rates: %w", err)
	}

	now := time.Now()
	for _, rate := range rates.Rates {
		err := s.store.Put(ctx, rate.Currency, rate.Rate)
		if err != nil {
			return fmt.Errorf("error setting exchange rate: %w", err)
		}

		s.logger.Info("currency: %s, rate: %f", rate.Currency, rate.Rate)

		historicalRateKey := historicalRateKey(now, rate.Currency)
		err = s.store.Put(ctx, historicalRateKey, rate.Rate)
		if err != nil {
			return fmt.Errorf("error setting historical exchange rate, key: %s %w", historicalRateKey, err)
		}
	}

	newTime := time.Now()
	err = s.store.Put(ctx, ExchangeRateDateKey, newTime)

	if err != nil {
		return fmt.Errorf("error setting refresh date %w", err)
	}

	s.logger.Info("new exchange rates are set")
	return nil
}

func (s *updaterService) needsRefresh(ctx context.Context) bool {
	var date time.Time
	exists, err := s.store.Get(ctx, ExchangeRateDateKey, &date)
	if err != nil {
		s.logger.Info("error fetching date")

		return true
	}

	if !exists {
		s.logger.Info("date doesn't exist")

		return true
	}

	comparisonDate := time.Now().Add(-ExchangeRateRefresh)

	if date.Before(comparisonDate) {
		s.logger.Info("comparison date was more than 8 hours ago")

		return true
	}

	return false
}

func (s *updaterService) EnsureHistoricalExchangeRates(ctx context.Context) error {
	if !s.historicalRatesNeedRefresh(ctx) {
		return nil
	}

	startDate := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

	s.logger.Info("requesting historical exchange rates")
	rates, err := s.provider.FetchHistoricalExchangeRates(ctx, startDate)
	if err != nil {
		return fmt.Errorf("error getting historical currency exchange rates: %w", err)
	}

	// the API doesn't return rates for weekends and public holidays at the time of writing this,
	// so we fill in the missing values using values from previously available days
	rates, err = fillInGapDays(rates, s.clock)
	if err != nil {
		return fmt.Errorf("error filling in gaps: %w", err)
	}

	keyValues := make(map[string]float64)
	for _, dayRates := range rates {
		for _, rate := range dayRates.Rates {
			key := historicalRateKey(dayRates.Day, rate.Currency)
			keyValues[key] = rate.Rate
		}
	}

	err = s.store.PutBatch(ctx, keyValues)
	if err != nil {
		return fmt.Errorf("error setting historical exchange rates: %w", err)
	}

	newTime := s.clock.Now()
	err = s.store.Put(ctx, HistoricalExchangeRateDateKey, newTime)
	if err != nil {
		return fmt.Errorf("error setting historical refresh date %w", err)
	}

	s.logger.Info("stored %d day-currency combinations of historical exchange rates", len(keyValues))
	return nil
}

func (s *updaterService) historicalRatesNeedRefresh(ctx context.Context) bool {
	var date time.Time
	exists, err := s.store.Get(ctx, HistoricalExchangeRateDateKey, &date)
	if err != nil {
		s.logger.Info("historicalRatesNeedRefresh error fetching date")

		return true
	}

	if !exists {
		s.logger.Info("historicalRatesNeedRefresh date doesn't exist")

		return true
	}

	comparisonDate := s.clock.Now().Add(-24 * time.Hour)

	if date.Before(comparisonDate) {
		s.logger.Info("historicalRatesNeedRefresh comparison date was more than threshold")

		return true
	}

	return false
}

func historicalRateKey(time time.Time, currency string) string {
	return time.Format(YMDLayout) + "-" + currency
}

func fillInGapDays(historicalContent []Rates, clock clock.Clock) ([]Rates, error) {
	var startDate time.Time
	endDate := clock.Now()
	dailyRates := make(map[string]Rates)

	for _, dayRates := range historicalContent {
		date := dayRates.Day
		if startDate.IsZero() || startDate.After(date) {
			startDate = date
		}
		dailyRates[date.Format(YMDLayout)] = dayRates
	}

	if startDate.IsZero() {
		return nil, fmt.Errorf("fillInGapDays, no valid data provided - startDate")
	}

	lastDay := startDate
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		if _, ok := dailyRates[date.Format(YMDLayout)]; ok {
			lastDay = date
			continue
		}

		gapContent := Rates{
			Day:   date,
			Rates: dailyRates[lastDay.Format(YMDLayout)].Rates,
		}

		historicalContent = append(historicalContent, gapContent)
	}

	return historicalContent, nil
}
