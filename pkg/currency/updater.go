package currency

import (
	"encoding/xml"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
	"time"
)

type UpdaterService struct {
	logger mon.Logger
	http   http.Client
	redis  redis.Client
}

func NewUpdater(config cfg.Config, logger mon.Logger) *UpdaterService {
	redisClient := redis.GetClient(config, logger, redis.DefaultClientName)
	httpClient := http.NewHttpClient(logger)

	return NewUpdaterWithInterfaces(logger, redisClient, httpClient)
}

func NewUpdaterWithInterfaces(logger mon.Logger, redisClient redis.Client, httpClient http.Client) *UpdaterService {
	return &UpdaterService{
		logger: logger,
		redis:  redisClient,
		http:   httpClient,
	}
}

func (service *UpdaterService) EnsureRecentExchangeRates() error {
	dateString, err := service.redis.Get(ExchangeRateDateKey)

	needsRefresh := false

	if err != nil {
		service.logger.Info("CurrencyUpdaterService: Error fetching redis key, refetching exchange rates")
		needsRefresh = true
	}

	if len(dateString) == 0 {
		service.logger.Info("CurrencyUpdaterService: Date string was zero length, refetching exchange rates")
		needsRefresh = true
	}

	date, err := time.Parse(ExchangeRateDateFormat, dateString)
	comparisonDate := time.Now().Add(-time.Duration(RefreshAfterHours) * time.Hour)

	if err != nil {
		service.logger.Info("CurrencyUpdaterService: Error parsing comparison date, refetching exchange rates")
		needsRefresh = true
	}

	if err == nil && date.Before(comparisonDate) {
		service.logger.Info("CurrencyUpdaterService: Comparison date was more than 8 hours ago, refetching exchange rates")
		needsRefresh = true
	}

	if !needsRefresh {
		return nil
	}

	service.logger.Info("CurrencyUpdaterService: Requesting exchange rates")
	request := http.NewRequest(ExchangeRateUrl)
	response, err := service.http.Get(request)

	if err != nil {
		service.logger.Error(err, "CurrencyUpdaterService: Error while requesting exchange rates")
		return err
	}

	exchangeRateResult := ExchangeResponse{}
	err = xml.Unmarshal([]byte(response), &exchangeRateResult)

	if err != nil {
		service.logger.Error(err, "CurrencyUpdaterService: Error while unmarshalling exchange rates")
		return err
	}

	for _, Cube := range exchangeRateResult.Body.Content.Rates {
		err := service.redis.HSet(ExchangeRateDataKey, Cube.Currency, Cube.Rate)

		if err != nil {
			service.logger.Error(err, "CurrencyUpdaterService: Error while setting exchange rate")
			return err
		}

		service.logger.Infof("CurrencyUpdaterService: Currency: %s, Rate: %f", Cube.Currency, Cube.Rate)
	}

	newTime := time.Now().Format(ExchangeRateDateFormat)
	err = service.redis.Set(ExchangeRateDateKey, newTime, time.Duration(24)*time.Hour)
	if err != nil {
		service.logger.Error(err, "CurrencyUpdaterService: Failed setting refresh date")
	}

	service.logger.Info("CurrencyUpdaterService: New exchange rates are set")

	return nil
}
