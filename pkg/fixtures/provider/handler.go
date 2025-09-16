package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	FixturesProviderFactory func(ctx context.Context, config cfg.Config, logger log.Logger, metadata *appctx.Metadata) (FixturesProvider, error)
	FixturesProvider        interface {
		Provide(ctx context.Context) (any, error)
	}
)

var (
	fixturesProviderFactories map[string]FixturesProviderFactory
	fixturesProviderLck       sync.Mutex
)

func AddFixtureProviderFactory(storeName string, factory FixturesProviderFactory) {
	fixturesProviderLck.Lock()
	defer fixturesProviderLck.Unlock()

	if fixturesProviderFactories == nil {
		fixturesProviderFactories = map[string]FixturesProviderFactory{}
	}

	fixturesProviderFactories[storeName] = factory
}

func NewFixturesProviderHandler(ctx context.Context, config cfg.Config, logger log.Logger) (httpserver.HandlerWithoutInput, error) {
	metadata, err := appctx.ProvideMetadata(ctx)
	if err != nil {
		return nil, err
	}

	fixturesProviderLck.Lock()
	defer fixturesProviderLck.Unlock()
	fixturesProviders := map[string]FixturesProvider{}

	for name := range fixturesProviderFactories {
		if fixturesProviders[name], err = fixturesProviderFactories[name](ctx, config, logger, metadata); err != nil {
			return nil, fmt.Errorf("failed to create fixtures provider %s: %w", name, err)
		}
	}

	return &fixturesProviderHandler{
		fixturesProviders: fixturesProviders,
	}, nil
}

type fixturesProviderHandler struct {
	fixturesProviders map[string]FixturesProvider
}

func (d *fixturesProviderHandler) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	var store string
	var ok bool
	if store, ok = request.Params.Get("store"); !ok {
		return nil, fmt.Errorf("missing store parameter")
	}

	var handler FixturesProvider
	if handler, ok = d.fixturesProviders[store]; !ok {
		return nil, fmt.Errorf("no fixtures provider found for store %s", store)
	}

	var data any
	var err error
	if data, err = handler.Provide(ctx); err != nil {
		return nil, err
	}

	return httpserver.NewJsonResponse(data), nil
}
