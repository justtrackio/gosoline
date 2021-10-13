package uniqueid

import (
	"context"
	"fmt"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	GeneratorTypeMemory    = "memory"
	GeneratorTypeHttp      = "http"
	GeneratorTypeSonyFlake = "sonyflake"
	ConfigGeneratorType    = "unique_id.type"
	ConfigMachineId        = "unique_id.machine_id"
)

//go:generate mockery --name Generator
type Generator interface {
	NextId(ctx context.Context) (*int64, error)
}

var g = struct {
	sync.Mutex
	instance Generator
}{}

func ProvideGenerator(ctx context.Context, config cfg.Config, logger log.Logger) (Generator, error) {
	g.Lock()
	defer g.Unlock()

	if g.instance != nil {
		return g.instance, nil
	}

	var err error
	if g.instance, err = NewGenerator(ctx, config, logger); err != nil {
		return nil, err
	}

	return g.instance, nil
}

type GeneratorSettings struct {
	Type string `cfg:"type"`
}

func NewGenerator(ctx context.Context, config cfg.Config, logger log.Logger) (Generator, error) {
	generatorType := config.GetString("unique_id.type")

	switch generatorType {
	case GeneratorTypeMemory:
		return NewGeneratorMemory(ctx, config, logger)
	case GeneratorTypeHttp:
		return NewGeneratorHttp(ctx, config, logger)
	case GeneratorTypeSonyFlake:
		return NewGeneratorSonyFlake(ctx, config, logger)
	default:
		return nil, fmt.Errorf("invalid generator type: %s", generatorType)
	}
}
