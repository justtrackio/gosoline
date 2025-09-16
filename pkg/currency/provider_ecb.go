package currency

import (
	"context"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/http"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	ECBProviderName              = "ecb"
	ExchangeRateECBUrl           = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
	HistoricalExchangeRateECBUrl = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml"
)

type Body struct {
	Content Content `xml:"Cube"`
}

type Sender struct {
	Name string `xml:"name"`
}

type ExchangeResponse struct {
	Subject string `xml:"subject"`
	Sender  Sender `xml:"Sender"`
	Body    Body   `xml:"Cube"`
}

type HistoricalBody struct {
	Content []Content `xml:"Cube"`
}

type ECBProviderHistoricalExchangeResponse struct {
	Subject string         `xml:"subject"`
	Sender  Sender         `xml:"Sender"`
	Body    HistoricalBody `xml:"Cube"`
}

func newECBProvider(_ context.Context, logger log.Logger, http http.Client, _ ProviderSettings) Provider {
	return NewECBProviderWithInterfaces(logger, http)
}

func NewECBProviderWithInterfaces(logger log.Logger, http http.Client) Provider {
	return &ecbProvider{logger, http}
}

type ecbProvider struct {
	logger log.Logger
	http   http.Client
}

func (f *ecbProvider) Name() string {
	return ECBProviderName
}

func (f *ecbProvider) GetPriority() int {
	return -1 // negative value to always use this provider first
}

func (f *ecbProvider) FetchLatestRates(ctx context.Context) ([]Rate, error) {
	request := f.http.NewRequest().WithUrl(ExchangeRateECBUrl)
	response, err := f.http.Get(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error requesting ECB rates: %w", err)
	}

	exchangeRateResult := ExchangeResponse{}
	err = xml.Unmarshal(response.Body, &exchangeRateResult)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling ECB rates: %w", err)
	}

	return exchangeRateResult.Body.Content.Rates, nil
}

func (f *ecbProvider) FetchHistoricalRates(ctx context.Context, dates []time.Time) (map[time.Time][]Rate, error) {
	request := f.http.NewRequest().WithUrl(HistoricalExchangeRateECBUrl)
	response, err := f.http.Get(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("error requesting historical ECB rates: %w", err)
	}

	exchangeRateResult := ECBProviderHistoricalExchangeResponse{}
	err = xml.Unmarshal(response.Body, &exchangeRateResult)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling historical ECB rates: %w", err)
	}

	ecbRatesByDate := make(map[string][]Rate, len(exchangeRateResult.Body.Content))
	for _, dayRates := range exchangeRateResult.Body.Content {
		date, err := dayRates.GetTime()
		if err != nil {
			return nil, fmt.Errorf("error parsing time in historical exchange rates: %w", err)
		}

		ecbRatesByDate[date.Format(time.DateOnly)] = dayRates.Rates
	}

	result := make(map[time.Time][]Rate)
	for _, date := range dates {
		if rates, ok := ecbRatesByDate[date.Format(time.DateOnly)]; ok {
			result[date] = rates
		}
	}

	return result, nil
}
