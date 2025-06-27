package currency_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/currency"
	kvStoreMock "github.com/justtrackio/gosoline/pkg/kvstore/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/stretchr/testify/suite"
)

type serviceTestSuite struct {
	suite.Suite
	ctx context.Context

	logger logMocks.LoggerMock
	store  *kvStoreMock.KvStore[float64]
	clock  clock.FakeClock

	service currency.Service
}

func TestService(t *testing.T) {
	suite.Run(t, new(serviceTestSuite))
}

func (s *serviceTestSuite) SetupTest() {
	s.ctx = context.Background()

	s.logger = logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	s.store = kvStoreMock.NewKvStore[float64](s.T())
	s.clock = clock.NewFakeClockAt(time.Date(2021, 1, 3, 4, 20, 33, 0, time.UTC))

	s.service = currency.NewWithInterfaces(s.store, s.clock)
}

func (s *serviceTestSuite) TestToEur_Calculation() {
	s.mockCurrencyStoreGet(currency.Usd, 1.09, true)

	valueUsd := 1.09
	valueEur := 1.0

	converted, err := s.service.ToEur(s.ctx, valueUsd, currency.Usd)

	s.NoError(err)
	s.Equal(valueEur, converted)
}

func (s *serviceTestSuite) TestToUsd_Calculation() {
	s.mockCurrencyStoreGet(currency.Usd, 1.09, true)

	valueUsd := 1.09
	valueEur := 1.0

	converted, err := s.service.ToUsd(s.ctx, valueEur, currency.Eur)

	s.NoError(err)
	s.Equal(valueUsd, converted)
}

func (s *serviceTestSuite) TestHasCurrency() {
	s.store.EXPECT().Contains(s.ctx, currency.Usd).Return(true, nil).Once()

	hasCurrency, err := s.service.HasCurrency(s.ctx, currency.Usd)

	s.NoError(err)
	s.True(hasCurrency)
}

func (s *serviceTestSuite) TestHasCurrencyAtDate() {
	s.store.EXPECT().Contains(s.ctx, "2021-01-02-USD").Return(true, nil).Once()

	hasCurrency, err := s.service.HasCurrencyAtDate(s.ctx, currency.Usd, s.clock.Now().AddDate(0, 0, -1))

	s.NoError(err)
	s.True(hasCurrency)
}

func (s *serviceTestSuite) TestHasCurrencyAtDate_NotThere() {
	s.store.EXPECT().Contains(s.ctx, "2021-01-02-USD").Return(false, nil).Once()

	hasCurrency, err := s.service.HasCurrencyAtDate(s.ctx, currency.Usd, s.clock.Now().AddDate(0, 0, -1))

	s.NoError(err)
	s.False(hasCurrency)
}

func (s *serviceTestSuite) TestHasCurrencyAtDate_Error() {
	s.store.EXPECT().Contains(s.ctx, "2021-01-02-USD").Return(false, errors.New("lookup error")).Once()

	hasCurrency, err := s.service.HasCurrencyAtDate(s.ctx, currency.Usd, s.clock.Now().AddDate(0, 0, -1))

	s.EqualError(err, "CurrencyService: error looking up historic exchange rate for USD at 2021-01-02: lookup error")
	s.False(hasCurrency)
}

func (s *serviceTestSuite) TestToEurAtDate_Calculation() {
	s.mockCurrencyStoreGet("2021-01-02-USD", 1.09, true)

	valueUsd := 1.09
	valueEur := 1.0

	converted, err := s.service.ToEurAtDate(s.ctx, valueUsd, currency.Usd, s.clock.Now().AddDate(0, 0, -1))

	s.NoError(err)
	s.Equal(valueEur, converted)
}

func (s *serviceTestSuite) TestToEurAtDate_FallbackToPreviousDay() {
	s.mockCurrencyStoreGet("2021-01-03-USD", 0, false)
	s.mockCurrencyStoreGet("2021-01-02-USD", 1.09, true)

	valueUsd := 1.09
	valueEur := 1.0

	converted, err := s.service.ToEurAtDate(s.ctx, valueUsd, currency.Usd, s.clock.Now())

	s.NoError(err)
	s.Equal(valueEur, converted)
}

func (s *serviceTestSuite) TestToEurAtDate_DontFallbackToPreviousDay() {
	s.mockCurrencyStoreGet("2021-01-02-USD", 0, false)

	valueUsd := 1.09

	_, err := s.service.ToEurAtDate(s.ctx, valueUsd, currency.Usd, s.clock.Now().AddDate(0, 0, -1))

	s.EqualError(err, "CurrencyService: error parsing historic exchange rate for USD at 2021-01-02: CurrencyService: historic currency USD at 2021-01-02 not found")
}

func (s *serviceTestSuite) TestToEurAtDate_DateInFuture() {
	futureDate := s.clock.Now().AddDate(0, 0, 2)
	_, err := s.service.ToEurAtDate(s.ctx, 1, currency.Usd, futureDate)

	s.EqualError(err, "CurrencyService: requested date 2021-01-05T04:20:33Z is in the future")
}

func (s *serviceTestSuite) TestToUsdAtDate_Calculation() {
	s.mockCurrencyStoreGet("2021-01-02-USD", 1.09, true)

	valueUsd := 1.09
	valueEur := 1.0

	converted, err := s.service.ToUsdAtDate(s.ctx, valueEur, currency.Eur, s.clock.Now().AddDate(0, 0, -1))

	s.NoError(err)
	s.Equal(valueUsd, converted)
}

func (s *serviceTestSuite) TestToUsdAtDate_FallbackToPreviousDay() {
	s.mockCurrencyStoreGet("2021-01-03-USD", 0, false)
	s.mockCurrencyStoreGet("2021-01-02-USD", 1.09, true)

	valueUsd := 1.09
	valueEur := 1.0

	converted, err := s.service.ToUsdAtDate(s.ctx, valueEur, currency.Eur, s.clock.Now())

	s.NoError(err)
	s.Equal(valueUsd, converted)
}

func (s *serviceTestSuite) TestToUsdAtDate_ClockSkew() {
	s.mockCurrencyStoreGet("2021-01-03-USD", 2, true)

	got, err := s.service.ToUsdAtDate(s.ctx, 3.5, currency.Usd, s.clock.Now().Add(26*time.Hour))
	s.NoError(err)
	s.Equal(3.5, got)

	got, err = s.service.ToUsdAtDate(s.ctx, 12.12, "EUR", s.clock.Now().Add(59*time.Second))
	s.NoError(err)
	s.Equal(24.24, got)

	got, err = s.service.ToUsdAtDate(s.ctx, 23.23, "EUR", s.clock.Now().Add(61*time.Second))
	s.EqualError(err, "CurrencyService: requested date 2021-01-03T04:21:34Z is in the future")
	s.Equal(0.0, got)
}

func (s *serviceTestSuite) mockCurrencyStoreGet(key string, value float64, found bool) {
	s.store.EXPECT().Get(s.ctx, key, mdl.Box(0.0)).Run(func(ctx context.Context, key any, f *float64) {
		*f = value
	}).Return(found, nil).Once()
}
