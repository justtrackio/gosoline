package currency

import (
	"context"
	"encoding/xml"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

const (
	ExchangeRateRefresh    = 8 * time.Hour
	ExchangeRateUrl        = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
	ExchangeRateDateKey    = "currency_exchange_last_refresh"
	ExchangeRateDateFormat = time.RFC3339
)

type UpdaterService struct {
	logger mon.Logger
	http   http.Client
	store  kvstore.KvStore
}

func NewUpdater(config cfg.Config, logger mon.Logger) *UpdaterService {
	logger = logger.WithChannel("currency_updater_service")
	store := kvstore.NewConfigurableKvStore(config, logger, "currency")
	httpClient := http.NewHttpClient(config, logger)

	return NewUpdaterWithInterfaces(logger, store, httpClient)
}

func NewUpdaterWithInterfaces(logger mon.Logger, store kvstore.KvStore, httpClient http.Client) *UpdaterService {
	return &UpdaterService{
		logger: logger,
		store:  store,
		http:   httpClient,
	}
}

func (service *UpdaterService) EnsureRecentExchangeRates(ctx context.Context) error {
	if !service.needsRefresh(ctx) {
		return nil
	}

	service.logger.Info("refetching exchange rates")

	service.logger.Info("requesting exchange rates")
	rates, err := service.getCurrencyRates(ctx)

	if err != nil {
		service.logger.Error(err, "error getting currency exchange rates")
		return err
	}

	for _, rate := range rates {
		err := service.store.Put(ctx, rate.Currency, rate.Rate)

		if err != nil {
			service.logger.Error(err, "error setting exchange rate")
			return err
		}

		service.logger.Infof("currency: %s, rate: %f", rate.Currency, rate.Rate)
	}

	newTime := time.Now().Format(ExchangeRateDateFormat)
	err = service.store.Put(ctx, ExchangeRateDateKey, newTime)
	if err != nil {
		service.logger.Error(err, "error setting refresh date")
	}

	service.logger.Info("new exchange rates are set")

	return nil
}

func (service *UpdaterService) needsRefresh(ctx context.Context) bool {
	var dateString string
	exists, err := service.store.Get(ctx, ExchangeRateDateKey, &dateString)

	if err != nil {
		service.logger.Info("error fetching date")

		return true
	}

	if !exists || dateString == "" {
		service.logger.Info("date doesn't exist or is empty")

		return true
	}

	date, err := time.Parse(ExchangeRateDateFormat, dateString)

	if err != nil {
		service.logger.Info("error parsing date")

		return true
	}

	comparisonDate := time.Now().Add(ExchangeRateRefresh)

	if err == nil && date.Before(comparisonDate) {
		service.logger.Info("comparison date was more than 8 hours ago")

		return true
	}

	return false
}

func (service *UpdaterService) getCurrencyRates(ctx context.Context) ([]Rate, error) {
	request := service.http.NewRequest().WithUrl(ExchangeRateUrl)

	response, err := service.http.Get(ctx, request)

	if err != nil {
		service.logger.Error(err, "error requesting exchange rates")

		return nil, err
	}

	exchangeRateResult := ExchangeResponse{}
	err = xml.Unmarshal(response.Body, &exchangeRateResult)

	if err != nil {
		service.logger.Error(err, "error unmarshalling exchange rates")

		return nil, err
	}

	return exchangeRateResult.Body.Content.Rates, nil
}
