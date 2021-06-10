package currency

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mon"
	"time"
)

//go:generate mockery -name Service
type Service interface {
	HasCurrency(ctx context.Context, currency string) (bool, error)
	ToEur(ctx context.Context, value float64, from string) (float64, error)
	ToUsd(ctx context.Context, value float64, from string) (float64, error)
	ToCurrency(ctx context.Context, to string, value float64, from string) (float64, error)

	HasCurrencyAtDate(ctx context.Context, currency string, date time.Time) (bool, error)
	ToEurAtDate(ctx context.Context, value float64, from string, date time.Time) (float64, error)
	ToUsdAtDate(ctx context.Context, value float64, from string, date time.Time) (float64, error)
	ToCurrencyAtDate(ctx context.Context, to string, value float64, from string, date time.Time) (float64, error)
}

type currencyService struct {
	store kvstore.KvStore
	clock clock.Clock
}

func New(config cfg.Config, logger mon.Logger) (*currencyService, error) {
	store, err := kvstore.ProvideConfigurableKvStore(config, logger, kvStoreName)
	if err != nil {
		return nil, fmt.Errorf("can not create kvStore: %w", err)
	}

	return NewWithInterfaces(store, clock.Provider), nil
}

func NewWithInterfaces(store kvstore.KvStore, clock clock.Clock) *currencyService {
	return &currencyService{
		store: store,
		clock: clock,
	}
}

// returns whether we support converting a given currency or not and whether an error occurred or not
func (s *currencyService) HasCurrency(ctx context.Context, currency string) (bool, error) {
	if currency == "EUR" {
		return true, nil
	}

	return s.store.Contains(ctx, currency)
}

// returns the euro value for a given value and currency and nil if not error occurred. returns 0 and an error object otherwise.
func (s *currencyService) ToEur(ctx context.Context, value float64, from string) (float64, error) {
	if from == Eur {
		return value, nil
	}

	exchangeRate, err := s.getExchangeRate(ctx, from)

	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error parsing exchange rate: %w", err)
	}

	return value / exchangeRate, nil
}

// returns the us dollar value for a given value and currency and nil if not error occurred. returns 0 and an error object otherwise.
func (s *currencyService) ToUsd(ctx context.Context, value float64, from string) (float64, error) {
	if from == Usd {
		return value, nil
	}

	return s.ToCurrency(ctx, Usd, value, from)
}

// returns the value in the currency given in the to parameter for a given value and currency given in the from parameter and nil if not error occurred. returns 0 and an error object otherwise.
func (s *currencyService) ToCurrency(ctx context.Context, to string, value float64, from string) (float64, error) {
	if from == to {
		return value, nil
	}

	exchangeRate, err := s.getExchangeRate(ctx, to)

	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error parsing exchange rate: %w", err)
	}

	eur, err := s.ToEur(ctx, value, from)

	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error converting to eur: %w", err)
	}

	return eur * exchangeRate, nil
}

func (s *currencyService) getExchangeRate(ctx context.Context, to string) (float64, error) {
	var exchangeRate float64
	exists, err := s.store.Get(ctx, to, &exchangeRate)

	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error getting exchange rate: %w", err)
	} else if !exists {
		return 0, fmt.Errorf("CurrencyService: currency not found: %s", to)
	}

	return exchangeRate, nil
}

// looks up the exchange rate value for a given currency at a given date in the kvStore.
// if the date parameter is recent enough and a lookup for the given currency fails,
// function will return the lookup for the previous day
func (s *currencyService) getExchangeRateAtDate(ctx context.Context, currency string, date time.Time) (float64, error) {
	yesterday := s.clock.Now().AddDate(0, 0, -1)
	tomorrow := s.clock.Now().AddDate(0, 0, 1)
	var exchangeRate float64
	key := historicalRateKey(date, currency)

	exists, err := s.store.Get(ctx, key, &exchangeRate)

	if err != nil {
		return 0, fmt.Errorf("getExchangeRateAtDate: error getting exchange rate: %w", err)

	} else if !exists {
		if date.After(yesterday) && date.Before(tomorrow) {
			return s.getExchangeRateAtDate(ctx, currency, date.AddDate(0, 0, -1))
		}
		return 0, fmt.Errorf("getExchangeRateAtDate: currency not found: %s %s", currency, date)
	}

	return exchangeRate, nil
}

// returns whether we support converting a given currency at the given time or not and whether an error occurred or not
// if the date parameter is recent enough and a lookup for the given currency misses,
// function will return the lookup for the previous day
func (s *currencyService) HasCurrencyAtDate(ctx context.Context, currency string, date time.Time) (bool, error) {
	if currency == "EUR" {
		return true, nil
	}

	if date.After(time.Now()) {
		return false, fmt.Errorf("CurrencyService: requested date in the future")
	}

	yesterday := s.clock.Now().AddDate(0, 0, -1)
	tomorrow := s.clock.Now().AddDate(0, 0, 1)
	key := historicalRateKey(date, currency)
	exists, err := s.store.Contains(ctx, key)
	if err != nil {
		return exists, err

	} else if !exists && date.After(yesterday) && date.Before(tomorrow) {
		return s.HasCurrencyAtDate(ctx, currency, date.AddDate(0, 0, -1))
	}
	return exists, nil
}

// returns the euro value for a given value and currency at the given time and nil if not error occurred. returns 0 and an error object otherwise.
func (s *currencyService) ToEurAtDate(ctx context.Context, value float64, from string, date time.Time) (float64, error) {
	if from == Eur {
		return value, nil
	}

	if date.After(time.Now()) {
		return 0, fmt.Errorf("CurrencyService: requested date in the future")
	}

	exchangeRate, err := s.getExchangeRateAtDate(ctx, from, date)

	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error parsing exchange rate historically: %w", err)
	}

	return value / exchangeRate, nil
}

// returns the us dollar value for a given value and currency at the given time and nil if not error occurred. returns 0 and an error object otherwise.
func (s *currencyService) ToUsdAtDate(ctx context.Context, value float64, from string, date time.Time) (float64, error) {
	if from == Usd {
		return value, nil
	}

	return s.ToCurrencyAtDate(ctx, Usd, value, from, date)
}

// returns the value in the currency given in the to parameter for a given value and currency given in the from parameter and nil if not error occurred. returns 0 and an error object otherwise.
func (s *currencyService) ToCurrencyAtDate(ctx context.Context, to string, value float64, from string, date time.Time) (float64, error) {
	if from == to {
		return value, nil
	}

	if date.After(time.Now()) {
		return 0, fmt.Errorf("CurrencyService: requested date in the future")
	}

	exchangeRate, err := s.getExchangeRateAtDate(ctx, to, date)

	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error parsing exchange rate historically: %w", err)
	}

	eur, err := s.ToEurAtDate(ctx, value, from, date)

	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error converting to eur historically: %w", err)
	}

	return eur * exchangeRate, nil
}
