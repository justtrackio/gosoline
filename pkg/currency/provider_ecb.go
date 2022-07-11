package currency

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	ExchangeRateUrl           = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
	HistoricalExchangeRateUrl = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml"
)

func init() {
	AddProviderFactory("ecb", NewEcbProvider)
}

type ecbRate struct {
	Currency string  `xml:"currency,attr"`
	Rate     float64 `xml:"rate,attr"`
}

type ecbContent struct {
	Time  string    `xml:"time,attr"`
	Rates []ecbRate `xml:"Cube"`
}

type Body struct {
	Content ecbContent `xml:"Cube"`
}

type ecbSender struct {
	Name string `xml:"name"`
}

type ecbExchangeResponse struct {
	Subject string    `xml:"subject"`
	Sender  ecbSender `xml:"Sender"`
	Body    Body      `xml:"Cube"`
}

type ecbHistoricalBody struct {
	Content []ecbContent `xml:"Cube"`
}

type ecbHistoricalExchangeResponse struct {
	Subject string            `xml:"subject"`
	Sender  ecbSender         `xml:"Sender"`
	Body    ecbHistoricalBody `xml:"Cube"`
}

func (c ecbContent) GetTime() (time.Time, error) {
	t, err := time.Parse(YMDLayout, c.Time)

	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

type ecbProvider struct {
	http http.Client
}

func NewEcbProvider(_ context.Context, config cfg.Config, logger log.Logger) (Provider, error) {
	client := http.NewHttpClient(config, logger)

	return NewEcbProviderWithInterfaces(client), nil
}

func NewEcbProviderWithInterfaces(client http.Client) *ecbProvider {
	return &ecbProvider{
		http: client,
	}
}

func (p ecbProvider) FetchCurrentRates(ctx context.Context) (*Rates, error) {
	request := p.http.NewRequest().WithUrl(ExchangeRateUrl)

	response, err := p.http.Get(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error requesting exchange rates: %w", err)
	}

	exchangeRateResult := ecbExchangeResponse{}
	err = xml.Unmarshal(response.Body, &exchangeRateResult)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling exchange rates: %w", err)
	}

	day, err := exchangeRateResult.Body.Content.GetTime()
	if err != nil {
		return nil, fmt.Errorf("error parsing current day: %w", err)
	}

	rates := Rates{
		Day:   day,
		Rates: make([]Rate, len(exchangeRateResult.Body.Content.Rates)),
	}

	for i, rate := range exchangeRateResult.Body.Content.Rates {
		rates.Rates[i] = Rate(rate)
	}

	return &rates, nil
}

func (p ecbProvider) FetchHistoricalExchangeRates(ctx context.Context, startDate time.Time) ([]Rates, error) {
	request := p.http.NewRequest().WithUrl(HistoricalExchangeRateUrl)

	response, err := p.http.Get(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error requesting historical exchange rates: %w", err)
	}

	exchangeRateResult := ecbHistoricalExchangeResponse{}
	err = xml.Unmarshal(response.Body, &exchangeRateResult)

	if err != nil {
		return nil, fmt.Errorf("error unmarshalling historical exchange rates: %w", err)
	}

	rates := make([]Rates, 0)

	for _, content := range exchangeRateResult.Body.Content {
		day, err := content.GetTime()
		if err != nil {
			return nil, fmt.Errorf("could not parse historical timestamp: %w", err)
		}

		if day.Before(startDate) {
			continue
		}

		dayRates := Rates{
			Day:   day,
			Rates: make([]Rate, len(content.Rates)),
		}

		for i, rate := range content.Rates {
			dayRates.Rates[i] = Rate(rate)
		}

		rates = append(rates, dayRates)
	}

	return rates, nil
}
