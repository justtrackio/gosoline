package currency

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/log"
	"time"
)

const (
	ExchangeRateRefresh           = 8 * time.Hour
	ExchangeRateUrl               = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
	HistoricalExchangeRateUrl     = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml"
	ExchangeRateDateKey           = "currency_exchange_last_refresh"
	HistoricalExchangeRateDateKey = "currency_exchange_historical_last_refresh"
)

const YMDLayout = "2006-01-02"

//go:generate mockery -name UpdaterService
type UpdaterService interface {
	EnsureRecentExchangeRates(ctx context.Context) error
	EnsureHistoricalExchangeRates(ctx context.Context) error
}

type updaterService struct {
	logger log.Logger
	http   http.Client
	store  kvstore.KvStore
	clock  clock.Clock
}

func NewUpdater(config cfg.Config, logger log.Logger) (UpdaterService, error) {
	logger = logger.WithChannel("currency_updater_service")

	store, err := kvstore.ProvideConfigurableKvStore(config, logger, kvStoreName)
	if err != nil {
		return nil, fmt.Errorf("can not create kvStore: %w", err)
	}

	httpClient := http.NewHttpClient(config, logger)

	return NewUpdaterWithInterfaces(logger, store, httpClient, clock.Provider), nil
}

func NewUpdaterWithInterfaces(logger log.Logger, store kvstore.KvStore, httpClient http.Client, clock clock.Clock) UpdaterService {
	return &updaterService{
		logger: logger,
		store:  store,
		http:   httpClient,
		clock:  clock,
	}
}

func (s *updaterService) EnsureRecentExchangeRates(ctx context.Context) error {
	if !s.needsRefresh(ctx) {
		return nil
	}

	s.logger.Info("requesting exchange rates")
	rates, err := s.getCurrencyRates(ctx)

	if err != nil {
		return fmt.Errorf("error getting currency exchange rates: %w", err)
	}

	now := time.Now()
	for _, rate := range rates {
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

func (s *updaterService) getCurrencyRates(ctx context.Context) ([]Rate, error) {
	request := s.http.NewRequest().WithUrl(ExchangeRateUrl)

	response, err := s.http.Get(ctx, request)

	if err != nil {
		return nil, fmt.Errorf("error requesting exchange rates: %w", err)
	}

	exchangeRateResult := ExchangeResponse{}
	err = xml.Unmarshal(response.Body, &exchangeRateResult)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling exchange rates: %w", err)
	}

	return exchangeRateResult.Body.Content.Rates, nil
}

func (s *updaterService) EnsureHistoricalExchangeRates(ctx context.Context) error {
	if !s.historicalRatesNeedRefresh(ctx) {
		return nil
	}

	startDate := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

	s.logger.Info("requesting historical exchange rates")
	rates, err := s.fetchExchangeRates(ctx)

	if err != nil {
		return fmt.Errorf("error getting historical currency exchange rates: %w", err)
	}

	rates, err = filterOutOldExchangeRates(rates, startDate)
	if err != nil {
		return fmt.Errorf("error filtering out old rates: %w", err)
	}

	// the API doesn't return rates for weekends and public holidays at the time of writing this,
	// so we fill in the missing values using values from previously available days
	rates, err = fillInGapDays(rates, s.clock)
	if err != nil {
		return fmt.Errorf("error filling in gaps: %w", err)
	}

	keyValues := make(map[string]float64)
	for _, dayRates := range rates {
		date, err := dayRates.GetTime()
		if err != nil {
			return fmt.Errorf("error parsing time in historical exchange rates: %w", err)
		}

		for _, rate := range dayRates.Rates {
			key := historicalRateKey(date, rate.Currency)
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

func (s *updaterService) fetchExchangeRates(ctx context.Context) ([]Content, error) {
	request := s.http.NewRequest().WithUrl(HistoricalExchangeRateUrl)

	response, err := s.http.Get(ctx, request)

	if err != nil {
		return nil, fmt.Errorf("error requesting historical exchange rates: %w", err)
	}

	exchangeRateResult := HistoricalExchangeResponse{}
	err = xml.Unmarshal(response.Body, &exchangeRateResult)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling historical exchange rates: %w", err)
	}

	return exchangeRateResult.Body.Content, nil
}

func historicalRateKey(time time.Time, currency string) string {
	return time.Format("2006-01-02") + "-" + currency
}

func filterOutOldExchangeRates(rates []Content, earliestDate time.Time) (ret []Content, e error) {
	for _, dayRates := range rates {
		date, err := dayRates.GetTime()
		if err != nil {
			e = fmt.Errorf("filterOutOldExchangeRates error parsing time: %w", err)
			return
		}
		if !date.Before(earliestDate) {
			ret = append(ret, dayRates)
		}
	}
	return
}

func fillInGapDays(historicalContent []Content, clock clock.Clock) ([]Content, error) {
	var startDate time.Time
	var endDate = clock.Now()
	var dailyRates = make(map[string]Content)

	for _, dayRates := range historicalContent {
		date, err := dayRates.GetTime()
		if err != nil {
			return nil, fmt.Errorf("fillInGapDays error: %w", err)
		}
		if startDate.IsZero() || startDate.After(date) {
			startDate = date
		}
		dailyRates[date.Format(YMDLayout)] = dayRates
	}

	if startDate.IsZero() {
		return nil, fmt.Errorf("fillInGapDays, no valid data provided - startDate")
	}

	var lastDay = startDate
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		if _, ok := dailyRates[date.Format(YMDLayout)]; !ok {
			gapContent := dailyRates[lastDay.Format(YMDLayout)]
			gapContent.Time = date.Format(YMDLayout)
			historicalContent = append(historicalContent, gapContent)
		} else {
			lastDay = date
		}
	}

	return historicalContent, nil
}
