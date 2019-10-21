package currency

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
	"github.com/pkg/errors"
	"strconv"
	"sync"
	"time"
)

const RefreshAfterHours = 8
const ExchangeRateUrl = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
const ExchangeRateDataKey = "currency_exchange_rate"
const ExchangeRateDateKey = "currency_exchange_last_refresh"
const ExchangeRateDateFormat = time.RFC3339

//go:generate mockery -name Service
type Service interface {
	Currencies() ([]string, error)
	HasCurrency(currency string) (bool, error)
	ToEur(float64, string) (float64, error)
	ToUsd(float64, string) (float64, error)
	ToCurrency(string, float64, string) (float64, error)
}

type CurrencyService struct {
	redis      redis.Client
	currencies []string
	lck        sync.Mutex
}

func New(config cfg.Config, logger mon.Logger) *CurrencyService {
	redisClient := redis.GetClient(config, logger, redis.DefaultClientName)

	return NewWithInterfaces(redisClient)
}

func NewWithInterfaces(redisClient redis.Client) *CurrencyService {
	return &CurrencyService{
		redis: redisClient,
	}
}

func (service *CurrencyService) Currencies() ([]string, error) {
	service.lck.Lock()
	defer service.lck.Unlock()

	if len(service.currencies) > 0 {
		return service.currencies, nil
	}

	currencies, err := service.redis.HKeys(ExchangeRateDataKey)

	if err != nil {
		return nil, errors.WithMessage(err, "CurrencyService: error getting currency keys")
	}

	service.currencies = currencies

	return service.currencies, nil
}

func (service *CurrencyService) HasCurrency(currency string) (bool, error) {
	return service.redis.HExists(ExchangeRateDataKey, currency)
}

func (service *CurrencyService) ToEur(value float64, from string) (float64, error) {
	if from == Eur {
		return value, nil
	}

	exchangeRate, err := service.getExchangeRate(from)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error parsing exchange rate")
	}

	return value / exchangeRate, nil
}

func (service *CurrencyService) ToUsd(value float64, from string) (float64, error) {
	if from == Usd {
		return value, nil
	}

	return service.ToCurrency(Usd, value, from)
}

func (service *CurrencyService) ToCurrency(to string, value float64, from string) (float64, error) {
	if from == to {
		return value, nil
	}

	exchangeRate, err := service.getExchangeRate(to)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error parsing exchange rate")
	}

	eur, err := service.ToEur(value, from)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error converting to eur")
	}

	return eur * exchangeRate, nil
}

func (service *CurrencyService) getExchangeRate(to string) (float64, error) {
	exchangeRateString, err := service.redis.HGet(ExchangeRateDataKey, to)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error getting exchange rate")
	}

	return strconv.ParseFloat(exchangeRateString, 64)

}
