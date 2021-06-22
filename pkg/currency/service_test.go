package currency_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/currency"
	"github.com/applike/gosoline/pkg/http"
	httpMock "github.com/applike/gosoline/pkg/http/mocks"
	kvStoreMock "github.com/applike/gosoline/pkg/kvstore/mocks"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
	"time"
)

var response = `<?xml version="1.0" encoding="UTF-8"?>
<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
<gesmes:subject>Reference rates</gesmes:subject>
<gesmes:Sender>
<gesmes:name>European Central Bank</gesmes:name>
</gesmes:Sender>
<Cube>
<Cube time='2019-06-13'>
<Cube currency='USD' rate='1.1289'/>
<Cube currency='JPY' rate='122.44'/>
<Cube currency='BGN' rate='1.9558'/>
<Cube currency='CZK' rate='25.581'/>
<Cube currency='DKK' rate='7.4678'/>
<Cube currency='GBP' rate='0.88948'/>
<Cube currency='HUF' rate='322.00'/>
<Cube currency='PLN' rate='4.2574'/>
<Cube currency='RON' rate='4.7221'/>
<Cube currency='SEK' rate='10.6968'/>
<Cube currency='CHF' rate='1.1207'/>
<Cube currency='ISK' rate='141.50'/>
<Cube currency='NOK' rate='9.7720'/>
<Cube currency='HRK' rate='7.4128'/>
<Cube currency='RUB' rate='72.9275'/>
<Cube currency='TRY' rate='6.6343'/>
<Cube currency='AUD' rate='1.6336'/>
<Cube currency='BRL' rate='4.3429'/>
<Cube currency='CAD' rate='1.5021'/>
<Cube currency='CNY' rate='7.8144'/>
<Cube currency='HKD' rate='8.8375'/>
<Cube currency='IDR' rate='16135.37'/>
<Cube currency='ILS' rate='4.0530'/>
<Cube currency='INR' rate='78.4745'/>
<Cube currency='KRW' rate='1335.74'/>
<Cube currency='MXN' rate='21.6384'/>
<Cube currency='MYR' rate='4.7068'/>
<Cube currency='NZD' rate='1.7201'/>
<Cube currency='PHP' rate='58.556'/>
<Cube currency='SGD' rate='1.5423'/>
<Cube currency='THB' rate='35.250'/>
<Cube currency='ZAR' rate='16.7876'/>
</Cube>
</Cube>
</gesmes:Envelope>`
var historicalResponse = `<?xml version="1.0" encoding="UTF-8"?>
<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
   <gesmes:subject>Reference rates</gesmes:subject>
   <gesmes:Sender>
      <gesmes:name>European Central Bank</gesmes:name>
   </gesmes:Sender>
   <Cube>
      <Cube time="2021-05-26">
         <Cube currency="USD" rate="1.2229" />
         <Cube currency="BGN" rate="1.9558" />
      </Cube>
      <Cube time="2021-05-23">
         <Cube currency="USD" rate="1.2212" />
         <Cube currency="JPY" rate="132.97" />
      </Cube>
   </Cube>
</gesmes:Envelope>`
var historicalRateKey = "2021-01-02-USD"
var historicalRateDate = time.Date(2021, time.January, 2, 0, 0, 0, 0, time.Local)

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
	client := new(httpMock.Client)

	store.On("Get", mock.AnythingOfType("*context.emptyCtx"), currency.ExchangeRateDateKey, mock.AnythingOfType("*time.Time")).Run(func(args mock.Arguments) {
		ptr := args.Get(2).(*time.Time)
		*ptr = time.Now().AddDate(-1, 0, 0)
	}).Return(true, nil)
	store.On("Put", mock.AnythingOfType("*context.emptyCtx"), currency.ExchangeRateDateKey, mock.AnythingOfType("time.Time")).Return(nil)
	store.On("Put", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("string"), mock.AnythingOfType("float64")).Return(nil)

	r := &http.Response{
		Body: []byte(response),
	}

	client.On("NewRequest").Return(http.NewRequest(nil))
	client.On("Get", context.Background(), mock.AnythingOfType("*http.Request")).Return(r, nil)

	service := currency.NewUpdaterWithInterfaces(logger, store, client, clock.NewRealClock())

	err := service.EnsureRecentExchangeRates(context.Background())

	assert.NoError(t, err)

	store.AssertExpectations(t)
	client.AssertExpectations(t)
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
	client := new(httpMock.Client)
	fakeClock := clock.NewFakeClockAt(time.Date(2021, 05, 27, 0, 0, 0, 0, time.UTC))

	keyValyes := map[string]float64{
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
	store.On("PutBatch", mock.AnythingOfType("*context.emptyCtx"), keyValyes).Return(nil)
	store.On("Put", mock.AnythingOfType("*context.emptyCtx"), currency.HistoricalExchangeRateDateKey, fakeClock.Now()).Return(nil)

	r := &http.Response{
		Body: []byte(historicalResponse),
	}

	client.On("NewRequest").Return(http.NewRequest(nil))
	client.On("Get", context.Background(), mock.AnythingOfType("*http.Request")).Return(r, nil)

	service := currency.NewUpdaterWithInterfaces(logger, store, client, fakeClock)

	err := service.EnsureHistoricalExchangeRates(context.Background())

	assert.NoError(t, err)

	store.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestUpdaterService_EnsureHistoricalExchangeRatesTwoGapDaysAtEnd(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
	store := new(kvStoreMock.KvStore)
	client := new(httpMock.Client)
	fakeClock := clock.NewFakeClockAt(time.Date(2021, 05, 28, 1, 0, 0, 0, time.UTC))

	keyValyes := map[string]float64{
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
	store.On("PutBatch", mock.AnythingOfType("*context.emptyCtx"), keyValyes).Return(nil)
	store.On("Put", mock.AnythingOfType("*context.emptyCtx"), currency.HistoricalExchangeRateDateKey, fakeClock.Now()).Return(nil)

	r := &http.Response{
		Body: []byte(historicalResponse),
	}

	client.On("NewRequest").Return(http.NewRequest(nil))
	client.On("Get", context.Background(), mock.AnythingOfType("*http.Request")).Return(r, nil)

	service := currency.NewUpdaterWithInterfaces(logger, store, client, fakeClock)

	err := service.EnsureHistoricalExchangeRates(context.Background())

	assert.NoError(t, err)

	store.AssertExpectations(t)
	client.AssertExpectations(t)
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
