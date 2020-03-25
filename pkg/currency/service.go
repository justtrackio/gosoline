package currency

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/pkg/errors"
	"sync"
)

//go:generate mockery -name Service
type Service interface {
	HasCurrency(ctx context.Context, currency string) (bool, error)
	ToEur(ctx context.Context, value float64, from string) (float64, error)
	ToUsd(ctx context.Context, value float64, from string) (float64, error)
	ToCurrency(ctx context.Context, to string, value float64, from string) (float64, error)
}

type CurrencyService struct {
	store      kvstore.KvStore
	currencies []string
	lck        sync.Mutex
}

func New(config cfg.Config, logger mon.Logger) *CurrencyService {
	store := kvstore.NewConfigurableKvStore(config, logger, "currency")

	return NewWithInterfaces(store)
}

func NewWithInterfaces(store kvstore.KvStore) *CurrencyService {
	return &CurrencyService{
		store: store,
	}
}

// returns whether we support converting a given currency or not and whether an error occurred or not
func (s *CurrencyService) HasCurrency(ctx context.Context, currency string) (bool, error) {
	if currency == "EUR" {
		return true, nil
	}

	return s.store.Contains(ctx, currency)
}

// returns the euro value for a given value and currency and nil if not error occurred. returns 0 and an error object otherwise.
func (s *CurrencyService) ToEur(ctx context.Context, value float64, from string) (float64, error) {
	if from == Eur {
		return value, nil
	}

	exchangeRate, err := s.getExchangeRate(ctx, from)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error parsing exchange rate")
	}

	return value / exchangeRate, nil
}

// returns the us dollar value for a given value and currency and nil if not error occurred. returns 0 and an error object otherwise.
func (s *CurrencyService) ToUsd(ctx context.Context, value float64, from string) (float64, error) {
	if from == Usd {
		return value, nil
	}

	return s.ToCurrency(ctx, Usd, value, from)
}

// returns the value in the currency given in the to parameter for a given value and currency given in the from parameter and nil if not error occurred. returns 0 and an error object otherwise.
func (s *CurrencyService) ToCurrency(ctx context.Context, to string, value float64, from string) (float64, error) {
	if from == to {
		return value, nil
	}

	exchangeRate, err := s.getExchangeRate(ctx, to)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error parsing exchange rate")
	}

	eur, err := s.ToEur(ctx, value, from)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error converting to eur")
	}

	return eur * exchangeRate, nil
}

func (s *CurrencyService) getExchangeRate(ctx context.Context, to string) (float64, error) {
	var exchangeRate float64
	exists, err := s.store.Get(ctx, to, &exchangeRate)

	if err != nil {
		return 0, errors.WithMessage(err, "CurrencyService: error getting exchange rate")
	} else if !exists {
		return 0, errors.WithMessage(err, "CurrencyService: currency not found")
	}

	return exchangeRate, nil
}
