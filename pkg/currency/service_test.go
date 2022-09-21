package currency_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/currency"
	currencyMocks "github.com/justtrackio/gosoline/pkg/currency/mocks"
	kvStoreMock "github.com/justtrackio/gosoline/pkg/kvstore/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	historicalRateKey  = "2021-01-02-USD"
	historicalRateDate = time.Date(2021, time.January, 2, 0, 0, 0, 0, time.Local)
)

func TestCurrencyService_ToEur_Calculation(t *testing.T) {
	store := new(kvStoreMock.KvStore)

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), currency.ExchangeRateDateKey, mock.AnythingOfType("*time.Time")).Run(func(args mock.Arguments) {
		ptr := args.Get(2).(*time.Time)
		*ptr = time.Now()
	}).Return(true, nil)
	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), "USD", mock.AnythingOfType("*float64")).Run(func(args mock.Arguments) {
		f := args.Get(2).(*float64)
		*f = 1.09
	}).Return(true, nil)

	service := currency.NewWithInterfaces(store, clock.NewRealClock())

	valueUsd := 1.09
	valueEur := 1.0
	from := "USD"

	converted, err := service.ToEur(context.Background(), valueUsd, from)

	assert.NoError(t, err)
	assert.Equal(t, valueEur, converted)
}

func TestCurrencyService_ToUsd_Calculation(t *testing.T) {
	store := new(kvStoreMock.KvStore)

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), currency.ExchangeRateDateKey, mock.AnythingOfType("*time.Time")).Run(func(args mock.Arguments) {
		ptr := args.Get(2).(*time.Time)
		*ptr = time.Now()
	}).Return(true, nil)
	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), "USD", mock.AnythingOfType("*float64")).Run(func(args mock.Arguments) {
		ptr := args.Get(2).(*float64)
		*ptr = 1.09
	}).Return(true, nil)

	service := currency.NewWithInterfaces(store, clock.NewRealClock())

	valueUsd := 1.09
	valueEur := 1.0
	from := "EUR"

	converted, err := service.ToUsd(context.Background(), valueEur, from)

	assert.NoError(t, err)
	assert.Equal(t, valueUsd, converted)
}

func TestUpdaterService_EnsureRecentExchangeRates(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	store := new(kvStoreMock.KvStore)
	provider := new(currencyMocks.Provider)
	clk := clock.NewRealClock()
	today := clk.Now().Add(24 * time.Hour)

	provider.On("FetchCurrentRates", mock.AnythingOfType("*context.emptyCtx")).Return(&currency.Rates{
		Day: today,
		Rates: []currency.Rate{
			{
				Currency: "USD",
				Rate:     1.1,
			},
		},
	}, nil)

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), currency.ExchangeRateDateKey, mock.AnythingOfType("*time.Time")).Run(func(args mock.Arguments) {
		ptr := args.Get(2).(*time.Time)
		*ptr = time.Now().AddDate(-1, 0, 0)
	}).Return(true, nil)
	store.On("Put", mock.AnythingOfType("*context.emptyCtx"), currency.ExchangeRateDateKey, mock.AnythingOfType("time.Time")).Return(nil)
	store.On("Put", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("float64")).Return(nil)

	service := currency.NewUpdaterWithInterfaces(logger, clock.NewRealClock(), provider, store)

	err := service.EnsureRecentExchangeRates(context.Background())

	assert.NoError(t, err)

	store.AssertExpectations(t)
	provider.AssertExpectations(t)
}

func TestCurrencyService_HasCurrency(t *testing.T) {
	store := new(kvStoreMock.KvStore)

	store.On("Contains", mock.AnythingOfType("*context.emptyCtx"), "USD").Return(true, nil).Times(1)

	service := currency.NewWithInterfaces(store, clock.NewRealClock())

	hasCurrency, err := service.HasCurrency(context.Background(), "USD")

	assert.NoError(t, err)
	assert.True(t, hasCurrency)

	store.AssertExpectations(t)
}

func TestCurrencyService_HasCurrencyAtDate(t *testing.T) {
	store := new(kvStoreMock.KvStore)

	store.On("Contains", mock.AnythingOfType("*context.emptyCtx"), "2021-01-02-USD").Return(true, nil).Times(1)

	service := currency.NewWithInterfaces(store, clock.NewRealClock())

	date := time.Date(2021, time.January, 2, 0, 0, 0, 0, time.Local)
	hasCurrency, err := service.HasCurrencyAtDate(context.Background(), "USD", date)

	assert.NoError(t, err)
	assert.True(t, hasCurrency)

	store.AssertExpectations(t)
}

func TestCurrencyService_HasCurrencyAtDate_NotThere(t *testing.T) {
	store := new(kvStoreMock.KvStore)

	store.On("Contains", mock.AnythingOfType("*context.emptyCtx"), "2021-01-02-USD").Return(false, nil).Times(1)

	service := currency.NewWithInterfaces(store, clock.NewRealClock())

	date := time.Date(2021, time.January, 2, 0, 0, 0, 0, time.Local)
	hasCurrency, err := service.HasCurrencyAtDate(context.Background(), "USD", date)

	assert.NoError(t, err)
	assert.False(t, hasCurrency)

	store.AssertExpectations(t)
}

func TestCurrencyService_HasCurrencyAtDate_Error(t *testing.T) {
	store := new(kvStoreMock.KvStore)

	store.On("Contains", mock.AnythingOfType("*context.emptyCtx"), historicalRateKey).Return(false, errors.New("lookup error")).Times(1)

	service := currency.NewWithInterfaces(store, clock.NewRealClock())

	hasCurrency, err := service.HasCurrencyAtDate(context.Background(), "USD", historicalRateDate)

	assert.Error(t, err)
	assert.False(t, hasCurrency)

	store.AssertExpectations(t)
}

func TestCurrencyService_ToEurAtDate_Calculation(t *testing.T) {
	store := new(kvStoreMock.KvStore)

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), historicalRateKey, mock.AnythingOfType("*float64")).Run(func(args mock.Arguments) {
		f := args.Get(2).(*float64)
		*f = 1.09
	}).Return(true, nil)

	service := currency.NewWithInterfaces(store, clock.NewRealClock())

	valueUsd := 1.09
	valueEur := 1.0
	from := "USD"

	converted, err := service.ToEurAtDate(context.Background(), valueUsd, from, historicalRateDate)

	assert.NoError(t, err)
	assert.Equal(t, valueEur, converted)
}

func TestCurrencyService_ToEurAtDate_FallbackToPreviousDay(t *testing.T) {
	store := new(kvStoreMock.KvStore)
	fakeClock := clock.NewFakeClockAt(time.Date(2021, 1, 3, 1, 0, 0, 0, time.UTC))

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), "2021-01-03-USD", mock.AnythingOfType("*float64")).Return(false, nil)

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), historicalRateKey, mock.AnythingOfType("*float64")).Run(func(args mock.Arguments) {
		f := args.Get(2).(*float64)
		*f = 1.09
	}).Return(true, nil)

	service := currency.NewWithInterfaces(store, fakeClock)

	valueUsd := 1.09
	valueEur := 1.0
	from := "USD"

	converted, err := service.ToEurAtDate(context.Background(), valueUsd, from, historicalRateDate.AddDate(0, 0, 1))

	assert.NoError(t, err)
	assert.Equal(t, valueEur, converted)
}

func TestCurrencyService_ToEurAtDate_DontFallbackToPreviousDay(t *testing.T) {
	store := new(kvStoreMock.KvStore)
	fakeClock := clock.NewFakeClockAt(time.Date(2021, 1, 2, 1, 0, 0, 0, time.UTC))

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), "2021-01-05-USD", mock.AnythingOfType("*float64")).Return(false, nil)

	service := currency.NewWithInterfaces(store, fakeClock)

	valueUsd := 1.09
	from := "USD"

	_, err := service.ToEurAtDate(context.Background(), valueUsd, from, fakeClock.Now().AddDate(0, 0, 3))

	assert.Error(t, err)
}

func TestCurrencyService_ToEurAtDate_DateInFuture(t *testing.T) {
	store := new(kvStoreMock.KvStore)
	service := currency.NewWithInterfaces(store, clock.NewRealClock())

	from := "USD"
	futureDate := time.Now().AddDate(0, 0, 2)
	_, err := service.ToEurAtDate(context.Background(), 1, from, futureDate)

	assert.Error(t, err)
}

func TestCurrencyService_ToUsdAtDate_Calculation(t *testing.T) {
	store := new(kvStoreMock.KvStore)

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), historicalRateKey, mock.AnythingOfType("*float64")).Run(func(args mock.Arguments) {
		ptr := args.Get(2).(*float64)
		*ptr = 1.09
	}).Return(true, nil)

	service := currency.NewWithInterfaces(store, clock.NewRealClock())

	valueUsd := 1.09
	valueEur := 1.0
	from := "EUR"

	converted, err := service.ToUsdAtDate(context.Background(), valueEur, from, historicalRateDate)

	assert.NoError(t, err)
	assert.Equal(t, valueUsd, converted)
}

func TestCurrencyService_ToUsdAtDate_FallbackToPreviousDay(t *testing.T) {
	store := new(kvStoreMock.KvStore)
	fakeClock := clock.NewFakeClockAt(time.Date(2021, 1, 3, 1, 0, 0, 0, time.UTC))

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), "2021-01-03-USD", mock.AnythingOfType("*float64")).Return(false, nil)

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), historicalRateKey, mock.AnythingOfType("*float64")).Run(func(args mock.Arguments) {
		ptr := args.Get(2).(*float64)
		*ptr = 1.09
	}).Return(true, nil)

	service := currency.NewWithInterfaces(store, fakeClock)

	valueUsd := 1.09
	valueEur := 1.0
	from := "EUR"

	converted, err := service.ToUsdAtDate(context.Background(), valueEur, from, historicalRateDate.AddDate(0, 0, 1))

	assert.NoError(t, err)
	assert.Equal(t, valueUsd, converted)
}

func TestUpdaterService_EnsureHistoricalExchangeRates(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	store := new(kvStoreMock.KvStore)
	fakeClock := clock.NewFakeClockAt(time.Date(2021, 5, 27, 0, 0, 0, 0, time.UTC))

	keyValues := map[string]float64{
		"2021-05-27-USD": 1.2229,
		"2021-05-27-BGN": 1.9558,
		"2021-05-26-USD": 1.2229,
		"2021-05-26-BGN": 1.9558,
		"2021-05-25-USD": 1.2212,
		"2021-05-25-JPY": 132.97,
		"2021-05-24-USD": 1.2212,
		"2021-05-24-JPY": 132.97,
		"2021-05-23-USD": 1.2212,
		"2021-05-23-JPY": 132.97,
	}

	provider := new(currencyMocks.Provider)
	provider.On("FetchHistoricalExchangeRates", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("time.Time")).Return([]currency.Rates{
		{
			Day: time.Date(2021, 5, 23, 0, 0, 0, 0, time.UTC),
			Rates: []currency.Rate{
				{
					Currency: "USD",
					Rate:     1.2212,
				},
				{
					Currency: "JPY",
					Rate:     132.97,
				},
			},
		},
		{
			Day: time.Date(2021, 5, 26, 0, 0, 0, 0, time.UTC),
			Rates: []currency.Rate{
				{
					Currency: "USD",
					Rate:     1.2229,
				},
				{
					Currency: "BGN",
					Rate:     1.9558,
				},
			},
		},
	}, nil)

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), currency.HistoricalExchangeRateDateKey, mock.AnythingOfType("*time.Time")).Return(false, nil)
	store.On("PutBatch", mock.AnythingOfType("*context.emptyCtx"), keyValues).Return(nil)
	store.On("Put", mock.AnythingOfType("*context.emptyCtx"), currency.HistoricalExchangeRateDateKey, fakeClock.Now()).Return(nil)

	service := currency.NewUpdaterWithInterfaces(logger, fakeClock, provider, store)

	err := service.EnsureHistoricalExchangeRates(context.Background())

	assert.NoError(t, err)

	store.AssertExpectations(t)
	provider.AssertExpectations(t)
}

func TestUpdaterService_EnsureHistoricalExchangeRatesTwoGapDaysAtEnd(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	store := new(kvStoreMock.KvStore)
	fakeClock := clock.NewFakeClockAt(time.Date(2021, 0o5, 28, 1, 0, 0, 0, time.UTC))

	provider := new(currencyMocks.Provider)
	provider.On("FetchHistoricalExchangeRates", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("time.Time")).Return([]currency.Rates{
		{
			Day: time.Date(2021, 5, 23, 0, 0, 0, 0, time.UTC),
			Rates: []currency.Rate{
				{
					Currency: "USD",
					Rate:     1.2212,
				},
				{
					Currency: "JPY",
					Rate:     132.97,
				},
			},
		},
		{
			Day: time.Date(2021, 5, 26, 0, 0, 0, 0, time.UTC),
			Rates: []currency.Rate{
				{
					Currency: "USD",
					Rate:     1.2229,
				},
				{
					Currency: "BGN",
					Rate:     1.9558,
				},
			},
		},
	}, nil)

	keyValues := map[string]float64{
		"2021-05-28-USD": 1.2229,
		"2021-05-28-BGN": 1.9558,
		"2021-05-27-USD": 1.2229,
		"2021-05-27-BGN": 1.9558,
		"2021-05-26-USD": 1.2229,
		"2021-05-26-BGN": 1.9558,
		"2021-05-25-USD": 1.2212,
		"2021-05-25-JPY": 132.97,
		"2021-05-24-USD": 1.2212,
		"2021-05-24-JPY": 132.97,
		"2021-05-23-USD": 1.2212,
		"2021-05-23-JPY": 132.97,
	}
	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), currency.HistoricalExchangeRateDateKey, mock.AnythingOfType("*time.Time")).Return(false, nil)
	store.On("PutBatch", mock.AnythingOfType("*context.emptyCtx"), keyValues).Return(nil)
	store.On("Put", mock.AnythingOfType("*context.emptyCtx"), currency.HistoricalExchangeRateDateKey, fakeClock.Now()).Return(nil)

	service := currency.NewUpdaterWithInterfaces(logger, fakeClock, provider, store)

	err := service.EnsureHistoricalExchangeRates(context.Background())

	assert.NoError(t, err)

	store.AssertExpectations(t)
	provider.AssertExpectations(t)
}

func Test_ToUsdAtDate_closenessMargin(t *testing.T) {
	fakeClock := clock.NewFakeClockAt(time.Date(2021, 1, 3, 1, 0, 0, 0, time.UTC))
	store := new(kvStoreMock.KvStore)
	store.On("Get", context.Background(), "2021-01-03-USD", mock.AnythingOfType("*float64")).Run(func(args mock.Arguments) {
		ptr := args.Get(2).(*float64)
		*ptr = 2
	}).Return(true, nil)
	service := currency.NewWithInterfaces(store, fakeClock)

	got, err := service.ToUsdAtDate(context.Background(), 3.5, "USD", fakeClock.Now().Add(26*time.Hour))
	assert.NoError(t, err)
	assert.Equal(t, 3.5, got)

	got, err = service.ToUsdAtDate(context.Background(), 12.12, "EUR", fakeClock.Now().Add(59*time.Second))
	assert.NoError(t, err)
	assert.Equal(t, 24.24, got)

	got, err = service.ToUsdAtDate(context.Background(), 23.23, "EUR", fakeClock.Now().Add(61*time.Second))
	assert.Equal(t, err, fmt.Errorf("CurrencyService: requested date in the future"))
	assert.Equal(t, 0.0, got)
}
