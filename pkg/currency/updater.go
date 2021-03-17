package currency

import (
	"context"
	"encoding/xml"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/http"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/tracing"
	"time"
)

const (
	ExchangeRateRefresh = 8 * time.Hour
	ExchangeRateUrl     = "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml"
	ExchangeRateDateKey = "currency_exchange_last_refresh"
)

//go:generate mockery -name UpdaterService
type UpdaterService interface {
	EnsureRecentExchangeRates(ctx context.Context) error
}

type updaterService struct {
	logger mon.Logger
	tracer tracing.Tracer
	http   http.Client
	store  kvstore.KvStore
}

func NewUpdater(config cfg.Config, logger mon.Logger) (UpdaterService, error) {
	logger = logger.WithChannel("currency_updater_service")

	tracer, err := tracing.ProvideTracer(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create tracer: %w", err)
	}

	store, err := kvstore.NewConfigurableKvStore(config, logger, "currency")
	if err != nil {
		return nil, fmt.Errorf("can not create kvStore: %w", err)
	}

	httpClient := http.NewHttpClient(config, logger)

	return NewUpdaterWithInterfaces(logger, tracer, store, httpClient), nil
}

func NewUpdaterWithInterfaces(logger mon.Logger, tracer tracing.Tracer, store kvstore.KvStore, httpClient http.Client) UpdaterService {
	return &updaterService{
		logger: logger,
		tracer: tracer,
		store:  store,
		http:   httpClient,
	}
}

func (s *updaterService) EnsureRecentExchangeRates(ctx context.Context) error {
	ctx, span := s.tracer.StartSpanFromContext(ctx, "currency-update-service")
	defer span.Finish()

	if !s.needsRefresh(ctx) {
		return nil
	}

	s.logger.Info("requesting exchange rates")
	rates, err := s.getCurrencyRates(ctx)

	if err != nil {
		s.logger.Error(err, "error getting currency exchange rates")
		return err
	}

	for _, rate := range rates {
		err := s.store.Put(ctx, rate.Currency, rate.Rate)

		if err != nil {
			s.logger.Error(err, "error setting exchange rate")
			return err
		}

		s.logger.Infof("currency: %s, rate: %f", rate.Currency, rate.Rate)
	}

	newTime := time.Now()
	err = s.store.Put(ctx, ExchangeRateDateKey, newTime)

	if err != nil {
		s.logger.Error(err, "error setting refresh date")
	}

	s.logger.Info("new exchange rates are set")
	return nil
}

func (s *updaterService) needsRefresh(ctx context.Context) bool {
	var date time.Time
	exists, err := s.store.Get(ctx, ExchangeRateDateKey, &date)

	if err != nil {
		s.logger.Info("error fetching date")

		return true
	}

	if !exists {
		s.logger.Info("date doesn't exist")

		return true
	}

	comparisonDate := time.Now().Add(-ExchangeRateRefresh)

	if date.Before(comparisonDate) {
		s.logger.Info("comparison date was more than 8 hours ago")

		return true
	}

	return false
}

func (s *updaterService) getCurrencyRates(ctx context.Context) ([]Rate, error) {
	request := s.http.NewRequest().WithUrl(ExchangeRateUrl)

	response, err := s.http.Get(ctx, request)

	if err != nil {
		s.logger.Error(err, "error requesting exchange rates")

		return nil, err
	}

	exchangeRateResult := ExchangeResponse{}
	err = xml.Unmarshal(response.Body, &exchangeRateResult)

	if err != nil {
		s.logger.Error(err, "error unmarshalling exchange rates")

		return nil, err
	}

	return exchangeRateResult.Body.Content.Rates, nil
}
