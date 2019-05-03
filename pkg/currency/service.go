package currency

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/redis"
	"strconv"
)

const RefreshAfterHours = 8
const ExchangeRateUrl = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
const ExchangeRateDataKey = "currency_exchange_rate"
const ExchangeRateDateKey = "currency_exchange_last_refresh"
const ExchangeRateDateFormat = "2006-01-02 15:04:05"

//go:generate mockery -name Service
type Service interface {
	ToEur(value float64, from string) float64
}

type CurrencyService struct {
	logger mon.Logger
	redis  redis.Client
}

func New(config cfg.Config, logger mon.Logger) *CurrencyService {
	redisClient := redis.GetClient(config, logger, redis.DefaultClientName)

	return NewWithInterfaces(logger, redisClient)
}

func NewWithInterfaces(logger mon.Logger, redisClient redis.Client) *CurrencyService {
	return &CurrencyService{
		logger: logger,
		redis:  redisClient,
	}
}

func (service *CurrencyService) ToEur(value float64, from string) float64 {
	if from == Eur {
		return value
	}

	exchangeRateString, err := service.redis.HGet(ExchangeRateDataKey, from)

	if err != nil {
		service.logger.Error(err, "CurrencyService: error getting exchange rate")
		return 0
	}

	exchangeRate, err := strconv.ParseFloat(exchangeRateString, 64)

	if err != nil {
		service.logger.Error(err, "CurrencyService: error parsing exchange rate")
		return 0
	}

	return value / exchangeRate
}
