package currency_test

import (
	"context"
	net "net/http"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/currency"
	"github.com/justtrackio/gosoline/pkg/http"
	httpMock "github.com/justtrackio/gosoline/pkg/http/mocks"
	kvStoreMock "github.com/justtrackio/gosoline/pkg/kvstore/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var (
	response = `<?xml version="1.0" encoding="UTF-8"?>
<gesmes:Envelope xmlns:gesmes="https://www.gesmes.org/xml/2002-08-01" xmlns="https://www.ecb.int/vocabulary/2002-08-01/eurofxref">
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
	historicalResponse = `<?xml version="1.0" encoding="UTF-8"?>
<gesmes:Envelope xmlns:gesmes="https://www.gesmes.org/xml/2002-08-01" xmlns="https://www.ecb.int/vocabulary/2002-08-01/eurofxref">
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
	openExchangeRatesApiResponse = `{
    "timestamp": 1698061860,
    "base": "EUR",
    "rates": {
        "ADA": 3.77,
        "USD": 1.5,
        "ZAR": 19.07,
        "NEWC": 42.42
    }
}`
	openExchangeRatesApiResponseHistorical_2021_05_27 = `{
    "timestamp": 1622073600,
    "base": "EUR",
    "rates": {
        "ADA": 1.80,
        "USD": 1.25
    }
}`
	openExchangeRatesApiResponseHistorical_2021_05_26 = `{
    "timestamp": 1621987200,
    "base": "EUR",
    "rates": {
        "ADA": 1.77,
        "USD": 1.20
    }
}`
	openExchangeRatesApiResponseHistorical_2021_05_25 = `{
    "timestamp": 1621900800,
    "base": "EUR",
    "rates": {
        "ADA": 1.70,
        "USD": 1.19
    }
}`
	openExchangeRatesApiResponseHistorical_2021_05_24 = `{
    "timestamp": 1621814400,
    "base": "EUR",
    "rates": {
        "ADA": 1.68,
        "USD": 1.18
    }
}`
	openExchangeRatesApiResponseHistorical_2021_05_23 = `{
    "timestamp": 1621728000,
    "base": "EUR",
    "rates": {
        "ADA": 1.65,
        "USD": 1.17
    }
}`
)

type updaterServiceTestSuite struct {
	suite.Suite
	ctx context.Context

	logger logMocks.LoggerMock
	store  *kvStoreMock.KvStore[float64]
	client *httpMock.Client
	clock  clock.FakeClock

	updater                         currency.UpdaterService
	updaterWithOpenExchangeRatesApi currency.UpdaterService
}

func TestNewUpdaterService(t *testing.T) {
	suite.Run(t, new(updaterServiceTestSuite))
}

func (s *updaterServiceTestSuite) SetupTest() {
	s.ctx = s.T().Context()

	s.logger = logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	s.store = kvStoreMock.NewKvStore[float64](s.T())
	s.client = httpMock.NewClient(s.T())
	s.clock = clock.NewFakeClockAt(time.Date(2021, 5, 27, 0, 0, 0, 0, time.UTC))

	ecbProvider := currency.NewECBProviderWithInterfaces(s.logger, s.client)
	openExchangeRatesApiProvider := currency.NewOpenExchangeRatesApiProviderWithInterfaces(s.logger, s.client, currency.ProviderSettings{ApiKey: "test", Priority: 1})

	s.updater = currency.NewUpdaterWithInterfaces(s.logger, s.store, s.client, s.clock, &currency.Settings{
		StartDate: time.Date(2021, 5, 23, 0, 0, 0, 0, time.UTC),
	}, []currency.Provider{ecbProvider})

	s.updaterWithOpenExchangeRatesApi = currency.NewUpdaterWithInterfaces(s.logger, s.store, s.client, s.clock, &currency.Settings{
		StartDate: time.Date(2021, 5, 23, 0, 0, 0, 0, time.UTC),
	}, []currency.Provider{ecbProvider, openExchangeRatesApiProvider})
}

func (s *updaterServiceTestSuite) TestEnsureRecentExchangeRates() {
	s.mockCurrencyStoreGetTime(currency.ExchangeRateDateKey, s.clock.Now().AddDate(-1, 0, 0), true)
	s.store.EXPECT().Put(s.ctx, currency.ExchangeRateDateKey, mock.AnythingOfType("float64")).Return(nil)
	s.store.EXPECT().Put(s.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("float64")).Return(nil)

	s.mockHttpRequest(response, net.StatusOK)

	err := s.updater.EnsureRecentExchangeRates(s.ctx)

	s.NoError(err)
}

func (s *updaterServiceTestSuite) TestEnsureRecentExchangeRatesWithOpenExchangeRatesApi() {
	s.mockCurrencyStoreGetTime(currency.ExchangeRateDateKey, s.clock.Now().AddDate(-1, 0, 0), true)
	s.store.EXPECT().Put(s.ctx, currency.ExchangeRateDateKey, mock.AnythingOfType("float64")).Return(nil)
	s.store.EXPECT().Put(s.ctx, mock.AnythingOfType("string"), mock.AnythingOfType("float64")).Return(nil)

	s.mockHttpRequest(response, net.StatusOK)
	s.mockHttpRequest(openExchangeRatesApiResponse, net.StatusOK)

	err := s.updaterWithOpenExchangeRatesApi.EnsureRecentExchangeRates(s.ctx)

	s.NoError(err)
}

func (s *updaterServiceTestSuite) TestEnsureHistoricalExchangeRates() {
	exchangeRates := map[string]float64{
		"2021-05-27-USD": 1.2229,
		"2021-05-27-JPY": 132.97,
		"2021-05-27-BGN": 1.9558,
		"2021-05-26-USD": 1.2229,
		"2021-05-26-JPY": 132.97,
		"2021-05-26-BGN": 1.9558,
		"2021-05-25-USD": 1.2212,
		"2021-05-25-JPY": 132.97,
		"2021-05-24-USD": 1.2212,
		"2021-05-24-JPY": 132.97,
		"2021-05-23-USD": 1.2212,
		"2021-05-23-JPY": 132.97,
	}
	s.mockCurrencyStoreGetTime(currency.HistoricalExchangeRateDateKey, time.Time{}, false)
	s.store.EXPECT().PutBatch(s.ctx, exchangeRates).Return(nil)
	s.store.EXPECT().Put(s.ctx, currency.HistoricalExchangeRateDateKey, float64(s.clock.Now().Unix())).Return(nil)

	s.mockHttpRequest(historicalResponse, net.StatusOK)
	err := s.updater.EnsureHistoricalExchangeRates(s.ctx)

	s.NoError(err)
}

func (s *updaterServiceTestSuite) TestEnsureHistoricalExchangeRatesWithOpenExchangeRatesApi() {
	exchangeRates := map[string]float64{
		"2021-05-27-USD": 1.25,
		"2021-05-27-JPY": 132.97,
		"2021-05-27-BGN": 1.9558,
		"2021-05-27-ADA": 1.80,
		"2021-05-26-USD": 1.2229,
		"2021-05-26-JPY": 132.97,
		"2021-05-26-BGN": 1.9558,
		"2021-05-26-ADA": 1.77,
		"2021-05-25-USD": 1.19,
		"2021-05-25-JPY": 132.97,
		"2021-05-25-ADA": 1.70,
		"2021-05-24-USD": 1.18,
		"2021-05-24-JPY": 132.97,
		"2021-05-24-ADA": 1.68,
		"2021-05-23-USD": 1.2212,
		"2021-05-23-JPY": 132.97,
		"2021-05-23-ADA": 1.65,
	}
	s.mockCurrencyStoreGetTime(currency.HistoricalExchangeRateDateKey, time.Time{}, false)
	s.store.EXPECT().PutBatch(s.ctx, exchangeRates).Return(nil)
	s.store.EXPECT().Put(s.ctx, currency.HistoricalExchangeRateDateKey, float64(s.clock.Now().Unix())).Return(nil)

	s.mockHttpRequest(historicalResponse, net.StatusOK)
	s.mockHttpRequest(openExchangeRatesApiResponseHistorical_2021_05_23, net.StatusOK)
	s.mockHttpRequest(openExchangeRatesApiResponseHistorical_2021_05_24, net.StatusOK)
	s.mockHttpRequest(openExchangeRatesApiResponseHistorical_2021_05_25, net.StatusOK)
	s.mockHttpRequest(openExchangeRatesApiResponseHistorical_2021_05_26, net.StatusOK)
	s.mockHttpRequest(openExchangeRatesApiResponseHistorical_2021_05_27, net.StatusOK)

	err := s.updaterWithOpenExchangeRatesApi.EnsureHistoricalExchangeRates(s.ctx)

	s.NoError(err)
}

func (s *updaterServiceTestSuite) TestEnsureHistoricalExchangeRatesTwoGapDaysAtEnd() {
	s.clock.Advance(time.Hour * 24)

	exchangeRates := map[string]float64{
		"2021-05-28-USD": 1.2229,
		"2021-05-28-JPY": 132.97,
		"2021-05-28-BGN": 1.9558,
		"2021-05-27-USD": 1.2229,
		"2021-05-27-JPY": 132.97,
		"2021-05-27-BGN": 1.9558,
		"2021-05-26-USD": 1.2229,
		"2021-05-26-JPY": 132.97,
		"2021-05-26-BGN": 1.9558,
		"2021-05-25-USD": 1.2212,
		"2021-05-25-JPY": 132.97,
		"2021-05-24-USD": 1.2212,
		"2021-05-24-JPY": 132.97,
		"2021-05-23-USD": 1.2212,
		"2021-05-23-JPY": 132.97,
	}
	s.mockCurrencyStoreGetTime(currency.HistoricalExchangeRateDateKey, time.Time{}, false)
	s.store.EXPECT().PutBatch(s.ctx, exchangeRates).Return(nil)
	s.store.EXPECT().Put(s.ctx, currency.HistoricalExchangeRateDateKey, float64(s.clock.Now().Unix())).Return(nil)

	s.mockHttpRequest(historicalResponse, net.StatusOK)

	err := s.updater.EnsureHistoricalExchangeRates(s.ctx)

	s.NoError(err)
}

func (s *updaterServiceTestSuite) mockCurrencyStoreGetTime(key string, value time.Time, found bool) {
	s.store.EXPECT().Get(s.ctx, key, new(float64)).Run(func(ctx context.Context, key any, f *float64) {
		*f = float64(value.Unix())
	}).Return(found, nil).Once()
}

func (s *updaterServiceTestSuite) mockHttpRequest(body string, statusCode int) {
	r := &http.Response{
		Body:       []byte(body),
		StatusCode: statusCode,
	}

	s.client.EXPECT().NewRequest().Return(http.NewRequest(nil)).Once()
	s.client.EXPECT().Get(s.ctx, mock.AnythingOfType("*http.Request")).Return(r, nil).Once()
}
