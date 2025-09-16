package currency

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	maxClockSkew = time.Minute
	oneDay       = time.Hour * 24
)

//go:generate go run github.com/vektra/mockery/v2 --name Service
type Service interface {
	HasCurrency(ctx context.Context, currency string) (bool, error)
	HasCurrencyAtDate(ctx context.Context, currency string, date time.Time) (bool, error)
	ToEur(ctx context.Context, value float64, fromCurrency string) (float64, error)
	ToEurAtDate(ctx context.Context, value float64, fromCurrency string, date time.Time) (float64, error)
	ToUsd(ctx context.Context, value float64, fromCurrency string) (float64, error)
	ToUsdAtDate(ctx context.Context, value float64, fromCurrency string, date time.Time) (float64, error)
	ToCurrency(ctx context.Context, toCurrency string, value float64, fromCurrency string) (float64, error)
	ToCurrencyAtDate(ctx context.Context, toCurrency string, value float64, fromCurrency string, date time.Time) (float64, error)
}

type currencyService struct {
	store kvstore.KvStore[float64]
	clock clock.Clock
}

func New(ctx context.Context, config cfg.Config, logger log.Logger) (Service, error) {
	store, err := kvstore.ProvideConfigurableKvStore[float64](ctx, config, logger, kvStoreName)
	if err != nil {
		return nil, fmt.Errorf("can not create currency kvStore: %w", err)
	}

	return NewWithInterfaces(store, clock.Provider), nil
}

func NewWithInterfaces(store kvstore.KvStore[float64], clock clock.Clock) Service {
	return &currencyService{
		store: store,
		clock: clock,
	}
}

// HasCurrency returns whether we support converting a given currency.
func (s *currencyService) HasCurrency(ctx context.Context, currency string) (bool, error) {
	if strings.EqualFold(currency, Eur) {
		return true, nil
	}

	exits, err := s.store.Contains(ctx, strings.ToUpper(currency))
	if err != nil {
		return false, fmt.Errorf("CurrencyService: error looking up exchange rate for %s: %w", currency, err)
	}

	return exits, nil
}

// HasCurrencyAtDate returns whether we support converting a given currency at the given time.
// We might fall back to yesterday's data if today's data is not yet up to date.
func (s *currencyService) HasCurrencyAtDate(ctx context.Context, currency string, date time.Time) (bool, error) {
	if strings.EqualFold(currency, Eur) {
		return true, nil
	}

	if date.After(s.clock.Now().Add(maxClockSkew)) {
		return false, fmt.Errorf("CurrencyService: requested date %s is in the future", date.Format(time.RFC3339))
	}

	key := historicalRateKey(date, strings.ToUpper(currency))
	exists, err := s.store.Contains(ctx, key)
	if err != nil {
		return false, fmt.Errorf("CurrencyService: error looking up historic exchange rate for %s at %s: %w", currency, date.Format(time.DateOnly), err)
	}

	if !exists && date.After(s.clock.Now().Add(-oneDay)) {
		return s.HasCurrencyAtDate(ctx, currency, date.AddDate(0, 0, -1))
	}

	return exists, nil
}

// ToEur returns the Euro value for a given value and currency.
func (s *currencyService) ToEur(ctx context.Context, value float64, fromCurrency string) (float64, error) {
	exchangeRate, err := s.getExchangeRateToEur(ctx, fromCurrency)
	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error parsing exchange rate for %s: %w", fromCurrency, err)
	}

	return value / exchangeRate, nil
}

// ToEurAtDate returns the Euro value for a given value and currency at the given time.
// We might fall back to yesterday's data if today's data is not yet up to date.
func (s *currencyService) ToEurAtDate(ctx context.Context, value float64, fromCurrency string, date time.Time) (float64, error) {
	if date.After(s.clock.Now().Add(maxClockSkew)) {
		return 0, fmt.Errorf("CurrencyService: requested date %s is in the future", date.Format(time.RFC3339))
	}

	exchangeRate, err := s.getExchangeRateToEurAtDate(ctx, fromCurrency, date)
	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error parsing historic exchange rate for %s at %s: %w", fromCurrency, date.Format(time.DateOnly), err)
	}

	return value / exchangeRate, nil
}

// ToUsd returns the US dollar value for a given value and currency.
func (s *currencyService) ToUsd(ctx context.Context, value float64, fromCurrency string) (float64, error) {
	return s.ToCurrency(ctx, Usd, value, fromCurrency)
}

// ToUsdAtDate returns the US dollar value for a given value and currency at the given time.
// We might fall back to yesterday's data if today's data is not yet up to date.
func (s *currencyService) ToUsdAtDate(ctx context.Context, value float64, fromCurrency string, date time.Time) (float64, error) {
	return s.ToCurrencyAtDate(ctx, Usd, value, fromCurrency, date)
}

// ToCurrency returns the value converted from one currency to another currency.
func (s *currencyService) ToCurrency(ctx context.Context, toCurrency string, value float64, fromCurrency string) (float64, error) {
	if strings.EqualFold(fromCurrency, toCurrency) {
		return value, nil
	}

	exchangeRate, err := s.getExchangeRateToEur(ctx, toCurrency)
	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error parsing exchange rate for %s: %w", toCurrency, err)
	}

	eur, err := s.ToEur(ctx, value, fromCurrency)
	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error converting %s to EUR: %w", fromCurrency, err)
	}

	return eur * exchangeRate, nil
}

// ToCurrencyAtDate returns the value converted from one currency to another currency at the given time.
// We might fall back to yesterday's data if today's data is not yet up to date.
func (s *currencyService) ToCurrencyAtDate(ctx context.Context, toCurrency string, value float64, fromCurrency string, date time.Time) (float64, error) {
	if strings.EqualFold(fromCurrency, toCurrency) {
		return value, nil
	}

	if date.After(s.clock.Now().Add(maxClockSkew)) {
		return 0, fmt.Errorf("CurrencyService: requested date %s is in the future", date.Format(time.RFC3339))
	}

	exchangeRate, err := s.getExchangeRateToEurAtDate(ctx, toCurrency, date)
	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error parsing historic exchange rate for %s at %s: %w", toCurrency, date.Format(time.DateOnly), err)
	}

	eur, err := s.ToEurAtDate(ctx, value, fromCurrency, date)
	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error converting historic %s to EUR at %s: %w", fromCurrency, date.Format(time.DateOnly), err)
	}

	return eur * exchangeRate, nil
}

// getExchangeRateToEurAtDate looks up the exchange rate value for a given currency in the kvStore.
func (s *currencyService) getExchangeRateToEur(ctx context.Context, currency string) (float64, error) {
	if strings.EqualFold(currency, Eur) {
		return 1, nil
	}

	var exchangeRate float64
	exists, err := s.store.Get(ctx, strings.ToUpper(currency), &exchangeRate)
	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error getting exchange rate for %s: %w", currency, err)
	}

	if !exists {
		return 0, fmt.Errorf("CurrencyService: currency %s not found", currency)
	}

	return exchangeRate, nil
}

// getExchangeRateToEurAtDate looks up the exchange rate value for a given currency at a given date in the kvStore.
// We might fall back to yesterday's data if today's data is not yet up to date.
func (s *currencyService) getExchangeRateToEurAtDate(ctx context.Context, currency string, date time.Time) (float64, error) {
	if strings.EqualFold(currency, Eur) {
		return 1, nil
	}

	var exchangeRate float64
	key := historicalRateKey(date, strings.ToUpper(currency))

	exists, err := s.store.Get(ctx, key, &exchangeRate)
	if err != nil {
		return 0, fmt.Errorf("CurrencyService: error getting historic exchange rate for %s at %s: %w", currency, date.Format(time.DateOnly), err)
	}

	if !exists {
		if date.After(s.clock.Now().Add(-oneDay)) {
			return s.getExchangeRateToEurAtDate(ctx, currency, date.AddDate(0, 0, -1))
		}

		return 0, fmt.Errorf("CurrencyService: historic currency %s at %s not found", currency, date.Format(time.DateOnly))
	}

	return exchangeRate, nil
}
