package currency

import (
	"context"
	"time"
)

//go:generate mockery --name Provider
type Provider interface {
	FetchCurrentRates(ctx context.Context) (*Rates, error)
	FetchHistoricalExchangeRates(ctx context.Context, startDate time.Time) ([]Rates, error)
}

type Currency string

type Rates struct {
	Day   time.Time
	Rates []Rate
}

type Rate struct {
	Currency string
	Rate     float64
}
