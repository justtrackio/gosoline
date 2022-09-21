package currency_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/currency"
	"github.com/justtrackio/gosoline/pkg/http"
	httpMocks "github.com/justtrackio/gosoline/pkg/http/mocks"
	"github.com/stretchr/testify/assert"
)

var currentResponse = `<?xml version="1.0" encoding="UTF-8"?>
<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
<gesmes:subject>Reference rates</gesmes:subject>
<gesmes:Sender>
<gesmes:name>European Central Bank</gesmes:name>
</gesmes:Sender>
<Cube>
<Cube time='2019-06-13'>
<Cube currency='USD' rate='1.1289'/>
<Cube currency='JPY' rate='122.44'/>
</Cube>
</Cube>
</gesmes:Envelope>`

func TestEcbProvider_FetchCurrentRates(t *testing.T) {
	req := http.NewRequest(nil)
	ctx := context.Background()

	resp := &http.Response{
		Body: []byte(currentResponse),
	}

	client := new(httpMocks.Client)

	client.On("NewRequest").Return(req)
	client.On("Get", ctx, req).Return(resp, nil)

	prv := currency.NewEcbProviderWithInterfaces(client)

	rates, err := prv.FetchCurrentRates(context.Background())

	expected := &currency.Rates{
		Day: time.Date(2019, 6, 13, 0, 0, 0, 0, time.UTC),
		Rates: []currency.Rate{
			{
				Currency: "USD",
				Rate:     1.1289,
			},
			{
				Currency: "JPY",
				Rate:     122.44,
			},
		},
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, rates)

	client.AssertExpectations(t)
}

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

func TestEcbProvider_FetchHistoricalExchangeRates(t *testing.T) {
	req := http.NewRequest(nil)
	ctx := context.Background()

	resp := &http.Response{
		Body: []byte(historicalResponse),
	}

	client := new(httpMocks.Client)

	client.On("NewRequest").Return(req)
	client.On("Get", ctx, req).Return(resp, nil)

	prv := currency.NewEcbProviderWithInterfaces(client)

	startDate := time.Date(2021, 5, 24, 0, 0, 0, 0, time.UTC)
	rates, err := prv.FetchHistoricalExchangeRates(context.Background(), startDate)

	expected := []currency.Rates{
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
	}

	assert.NoError(t, err)
	assert.Equal(t, expected, rates)

	client.AssertExpectations(t)
}
