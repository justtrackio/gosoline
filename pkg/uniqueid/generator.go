package uniqueid

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	GeneratorTypeMemory    = "memory"
	GeneratorTypeSrv       = "srv"
	GeneratorTypeSonyFlake = "sonyflake"
	ConfigGeneratorType    = "unique_id.type"
	ConfigMachineId        = "unique_id.machine_id"
)

//go:generate mockery --name Generator
type Generator interface {
	NextId(ctx context.Context) (*int64, error)
}

type generatorAppCtxKey string

var key = new(generatorAppCtxKey)

func WithGenerator(ctx context.Context, generator Generator) error {
	_, err := appctx.Provide(ctx, key, func() (interface{}, error) {
		return generator, nil
	})

	return err
}

func ProvideGenerator(ctx context.Context, config cfg.Config, logger log.Logger) (Generator, error) {
	generator, err := appctx.Provide(ctx, key, func() (interface{}, error) {
		return NewGenerator(config, logger)
	})
	if err != nil {
		return nil, err
	}

	return generator.(Generator), nil
}

type GeneratorSettings struct {
	Type string `cfg:"type"`
}

func NewGenerator(config cfg.Config, logger log.Logger) (Generator, error) {
	generatorType := config.GetString("unique_id.type")

	switch generatorType {
	case GeneratorTypeSrv:
		return NewGeneratorSrv(config, logger)
	case GeneratorTypeSonyFlake:
		return NewGeneratorSonyFlake(config, logger)
	default:
		return nil, fmt.Errorf("invalid generator type: %s", generatorType)
	}
}
