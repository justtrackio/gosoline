package currency_test

import (
	"context"
	"github.com/applike/gosoline/pkg/currency"
	"github.com/applike/gosoline/pkg/http"
	httpMock "github.com/applike/gosoline/pkg/http/mocks"
	loggerMock "github.com/applike/gosoline/pkg/mon/mocks"
	redisMock "github.com/applike/gosoline/pkg/redis/mocks"
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

func TestCurrencyService_ToEur_Calculation(t *testing.T) {
	redis := new(redisMock.Client)

	redis.On("Get", currency.ExchangeRateDateKey).Return(time.Now().Format(currency.ExchangeRateDateFormat), nil)
	redis.On("HGet", currency.ExchangeRateDataKey, "USD").Return("1.09", nil)

	service := currency.NewWithInterfaces(redis)

	valueUsd := 1.09
	valueEur := 1.0
	from := "USD"

	converted, err := service.ToEur(valueUsd, from)

	assert.NoError(t, err)
	assert.Equal(t, valueEur, converted)
}

func TestCurrencyService_ToUsd_Calculation(t *testing.T) {
	redis := new(redisMock.Client)

	redis.On("Get", currency.ExchangeRateDateKey).Return(time.Now().Format(currency.ExchangeRateDateFormat), nil)
	redis.On("HGet", currency.ExchangeRateDataKey, "USD").Return("1.09", nil)

	service := currency.NewWithInterfaces(redis)

	valueUsd := 1.09
	valueEur := 1.0
	from := "EUR"

	converted, err := service.ToUsd(valueEur, from)

	assert.NoError(t, err)
	assert.Equal(t, valueUsd, converted)
}

func TestUpdaterService_EnsureRecentExchangeRates(t *testing.T) {
	logger := loggerMock.NewLoggerMockedAll()
	redis := new(redisMock.Client)
	client := new(httpMock.Client)

	redis.On("Get", currency.ExchangeRateDateKey).Return(time.Now().AddDate(-1, 0, 0).Format(currency.ExchangeRateDateFormat), nil)
	redis.On("HSet", currency.ExchangeRateDataKey, mock.AnythingOfType("string"), mock.AnythingOfType("float64")).Return(nil)
	redis.On("Set", currency.ExchangeRateDateKey, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).Return(nil)

	r := &http.Response{
		Body: []byte(response),
	}

	client.On("NewRequest").Return(http.NewRequest(nil))
	client.On("Get", context.TODO(), mock.AnythingOfType("*http.Request")).Return(r, nil)

	service := currency.NewUpdaterWithInterfaces(logger, redis, client)

	err := service.EnsureRecentExchangeRates(context.TODO())

	assert.NoError(t, err)

	redis.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestCurrencyService_Currencies(t *testing.T) {
	redis := new(redisMock.Client)
	expectedCurrencies := []string{
		"EUR",
		"USD",
	}

	redis.On("HKeys", currency.ExchangeRateDataKey).Return(expectedCurrencies, nil).Times(1)

	service := currency.NewWithInterfaces(redis)

	currencies, err := service.Currencies()

	assert.NoError(t, err)
	assert.Equal(t, expectedCurrencies, currencies)

	// ask again, to ensure we used the applications cached currencies
	currencies, err = service.Currencies()

	assert.NoError(t, err)
	assert.Equal(t, expectedCurrencies, currencies)

	redis.AssertExpectations(t)
}

func TestCurrencyService_HasCurrency(t *testing.T) {
	redis := new(redisMock.Client)

	redis.On("HExists", currency.ExchangeRateDataKey, "USD").Return(true, nil).Times(1)

	service := currency.NewWithInterfaces(redis)

	hasCurrency, err := service.HasCurrency("USD")

	assert.NoError(t, err)
	assert.True(t, hasCurrency)

	redis.AssertExpectations(t)
}
