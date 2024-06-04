package calculator

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
)

var factories = map[string]HandlerFactory{}

type HandlerFactory func(ctx context.Context, config cfg.Config, logger log.Logger, calculatorSettings *CalculatorSettings) (Handler, error)

func RegisterHandlerFactory(name string, factory HandlerFactory) {
	if _, ok := factories[name]; ok {
		panic("factory with name " + name + " already exists")
	}

	factories[name] = factory
}

//go:generate mockery --name Handler
type Handler interface {
	GetMetrics(ctx context.Context) (metric.Data, error)
}
