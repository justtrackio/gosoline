package currency

import (
	"context"
	"encoding/json"
	"encoding/xml"
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
	ExchangeRateECBUrl            = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
	HistoricalExchangeRateECBUrl  = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml"
	ExchangeRateDateKey           = "currency_exchange_last_refresh"
	HistoricalExchangeRateDateKey = "currency_exchange_historical_last_refresh"
	ExchangeRateFxRatesUrl        = "https://api.fxratesapi.com/"
)

const (
	YMDLayout = "2006-01-02"
	// RatesAmount
	// We expect to have around 180 currencies in total (ECB + fxratesapi.com).
	// It's okay to exceed the limit, but it's nice to avoid unnecessary reallocations.
	RatesAmount = 180
)

type Settings struct {
	StartDate     time.Time `cfg:"start_date" default:"2015-01-01"`
	FxRatesApiKey string    `cfg:"fxratesapi_key"`
}

//go:generate go run github.com/vektra/mockery/v2 --name UpdaterService
type UpdaterService interface {
	EnsureRecentExchangeRates(ctx context.Context) error
	EnsureHistoricalExchangeRates(ctx context.Context) error
}

type updaterService struct {
	logger   log.Logger
	http     http.Client
	store    kvstore.KvStore[float64]
	clock    clock.Clock
	settings *Settings
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

	if settings.FxRatesApiKey == "" {
		logger.Warn("fxratesapi_key not set in config, fxratesapi.com rates will not be used")
	}

	return NewUpdaterWithInterfaces(logger, store, httpClient, clock.Provider, settings), nil
}

func NewUpdaterWithInterfaces(logger log.Logger, store kvstore.KvStore[float64], httpClient http.Client, clock clock.Clock, settings *Settings) UpdaterService {
	return &updaterService{
		logger:   logger,
		store:    store,
		http:     httpClient,
		clock:    clock,
		settings: settings,
	}
}

func (s *updaterService) EnsureRecentExchangeRates(ctx context.Context) error {
	logger := s.logger.WithContext(ctx)

	if !s.needsRefresh(ctx) {
		return nil
	}

	logger.Info("requesting exchange rates")
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

		logger.Info("currency: %s, rate: %f", rate.Currency, rate.Rate)

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

	logger.Info("new exchange rates are set")

	return nil
}

func (s *updaterService) needsRefresh(ctx context.Context) bool {
	logger := s.logger.WithContext(ctx)

	var dateUnix float64
	exists, err := s.store.Get(ctx, ExchangeRateDateKey, &dateUnix)
	if err != nil {
		logger.Info("error fetching date")

		return true
	}

	if !exists {
		logger.Info("date doesn't exist")

		return true
	}

	comparisonDate := s.clock.Now().Add(-ExchangeRateRefresh)

	date := time.Unix(int64(dateUnix), 0)

	if date.Before(comparisonDate) {
		logger.Info("comparison date was more than 8 hours ago")

		return true
	}

	return false
}

func (s *updaterService) getCurrencyRates(ctx context.Context) ([]Rate, error) {
	request := s.http.NewRequest().WithUrl(ExchangeRateECBUrl)

	response, err := s.http.Get(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error requesting exchange rates: %w", err)
	}

	exchangeRateResult := ExchangeResponse{}
	err = xml.Unmarshal(response.Body, &exchangeRateResult)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling exchange rates: %w", err)
	}

	rates := make([]Rate, 0, RatesAmount)
	rates = append(rates, exchangeRateResult.Body.Content.Rates...)

	ecbRateMap := make(map[string]float64)
	for _, r := range rates {
		ecbRateMap[r.Currency] = r.Rate
	}

	if s.settings.FxRatesApiKey == "" {
		s.logger.Warn("fxrates api key is not provided, skipping request to fxratesapi.com rates for recent data")

		return rates, nil
	}

	url := ExchangeRateFxRatesUrl + "latest?api_key=" + s.settings.FxRatesApiKey + "&base=EUR"

	fxRates, err := s.getFxRatesApiRates(ctx, url)
	if err != nil {
		s.logger.Warn("could not fetch fxratesapi.com rates: %s", err)

		return rates, nil
	}

	if fxRates == nil {
		return rates, nil
	}

	for currency, rate := range fxRates {
		if _, exists := ecbRateMap[currency]; !exists {
			rates = append(rates, Rate{Currency: currency, Rate: rate})
		}
	}

	return rates, nil
}

func (s *updaterService) EnsureHistoricalExchangeRates(ctx context.Context) error {
	logger := s.logger.WithContext(ctx)

	if !s.historicalRatesNeedRefresh(ctx) {
		return nil
	}

	logger.Info("requesting historical exchange rates")
	rates, err := s.fetchExchangeRates(ctx)
	if err != nil {
		return fmt.Errorf("error getting historical currency exchange rates: %w", err)
	}

	rates, err = filterOutOldExchangeRates(rates, s.settings.StartDate)
	if err != nil {
		return fmt.Errorf("error filtering out old rates: %w", err)
	}

	// the API doesn't return rates for weekends and public holidays at the time of writing this,
	// so we fill in the missing values using values from previously available days
	rates, err = fillInGapDays(rates, s.clock)
	if err != nil {
		return fmt.Errorf("error filling in gaps: %w", err)
	}

	if s.settings.FxRatesApiKey == "" {
		s.logger.Warn("fxrates api key is not provided, skipping request to fxratesapi.com rates for historical data")
	} else {
		rates, err = s.feedWithFxRatesApiRates(ctx, rates)
		if err != nil {
			s.logger.Warn("error during feeding historical data with fxrates data: %w", err)
		}
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

	logger.Info("updating %d historical exchange rates", len(keyValues))

	err = s.store.PutBatch(ctx, keyValues)
	if err != nil {
		return fmt.Errorf("error setting historical exchange rates: %w", err)
	}

	newTime := float64(s.clock.Now().Unix())
	err = s.store.Put(ctx, HistoricalExchangeRateDateKey, newTime)
	if err != nil {
		return fmt.Errorf("error setting historical refresh date %w", err)
	}

	logger.Info("stored %d day-currency combinations of historical exchange rates", len(keyValues))

	return nil
}

func (s *updaterService) getFxRatesApiRates(ctx context.Context, url string) (map[string]float64, error) {
	request := s.http.NewRequest().WithUrl(url)

	response, err := s.http.Get(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error requesting fxratesapi.com rates: %w", err)
	}

	var fxResp FxRatesApiResponse

	err = json.Unmarshal(response.Body, &fxResp)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling fxratesapi.com rates: %w", err)
	}

	if !fxResp.Success {
		return nil, fmt.Errorf("fxratesapi.com response not successful")
	}

	return fxResp.Rates, nil
}

func (s *updaterService) feedWithFxRatesApiRates(ctx context.Context, rates []Content) ([]Content, error) {
	for i, dayRates := range rates {
		date, err := dayRates.GetTime()
		if err != nil {
			return rates, fmt.Errorf("error parsing time in historical exchange rates: %w", err)
		}

		url := ExchangeRateFxRatesUrl + "historical?api_key=" + s.settings.FxRatesApiKey + "&base=EUR" + "&date=" + date.Format(YMDLayout)

		fxRates, err := s.getFxRatesApiRates(ctx, url)
		if err != nil {
			s.logger.Warn("could not fetch fxratesapi.com rates for the date %s: %s", date.Format(YMDLayout), err)
			// we don't fail the whole process if we can't get fxratesapi.com data for a specific day
			continue
		}

		present := make(map[string]bool)
		for _, rate := range dayRates.Rates {
			present[rate.Currency] = true
		}

		for currency, rate := range fxRates {
			if !present[currency] {
				dayRates.Rates = append(dayRates.Rates, Rate{Currency: currency, Rate: rate})
			}
		}

		rates[i] = dayRates
	}

	return rates, nil
}

func (s *updaterService) historicalRatesNeedRefresh(ctx context.Context) bool {
	logger := s.logger.WithContext(ctx)

	var dateUnix float64
	exists, err := s.store.Get(ctx, HistoricalExchangeRateDateKey, &dateUnix)
	if err != nil {
		logger.Info("historicalRatesNeedRefresh error fetching date")

		return true
	}

	if !exists {
		logger.Info("historicalRatesNeedRefresh date doesn't exist")

		return true
	}

	comparisonDate := s.clock.Now().Add(-24 * time.Hour)

	date := time.Unix(int64(dateUnix), 0)

	if date.Before(comparisonDate) {
		logger.Info("historicalRatesNeedRefresh comparison date was more than threshold")

		return true
	}

	return false
}

func (s *updaterService) fetchExchangeRates(ctx context.Context) ([]Content, error) {
	request := s.http.NewRequest().WithUrl(HistoricalExchangeRateECBUrl)

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
	endDate := clock.Now()
	dailyRates := make(map[string]Content)

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

	lastDay := startDate
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
