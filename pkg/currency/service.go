package currency

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

const RefreshAfterHours = 8
const ExchangeRateUrl = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
const ExchangeRateDataKey = "currency_exchange_rate"
const ExchangeRateDateKey = "currency_exchange_last_refresh"
const ExchangeRateDateFormat = time.RFC3339

//go:generate mockery -name Service
type Service interface {
	ToEur(value float64, from string) (float64, error)
}

type CurrencyService struct {
	redis redis.Client
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

func (service *CurrencyService) ToEur(value float64, from string) (float64, error) {
	if from == Eur {
		return value, nil
	}

	exchangeRateString, err := service.redis.HGet(ExchangeRateDataKey, from)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error getting exchange rate")
	}

	exchangeRate, err := strconv.ParseFloat(exchangeRateString, 64)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error parsing exchange rate")
	}

	return value / exchangeRate, nil
}
