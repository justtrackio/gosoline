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

// returns a slice of currencies we support converting and whether an error occurred querying them or not
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

	service.currencies = append([]string{"EUR"}, currencies...)

	return service.currencies, nil
}

// returns whether we support converting a given currency or not and whether an error occurred or not
func (service *CurrencyService) HasCurrency(currency string) (bool, error) {
	if currency == "EUR" {
		return true, nil
	}

	return service.redis.HExists(ExchangeRateDataKey, currency)
}

// returns the euro value for a given value and currency and nil if not error occurred. returns 0 and an error object otherwise.
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

// returns the us dollar value for a given value and currency and nil if not error occurred. returns 0 and an error object otherwise.
func (service *CurrencyService) ToUsd(value float64, from string) (float64, error) {
	if from == Usd {
		return value, nil
	}

	return service.ToCurrency(Usd, value, from)
}

// returns the value in the currency given in the to parameter for a given value and currency given in the from parameter and nil if not error occurred. returns 0 and an error object otherwise.
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
