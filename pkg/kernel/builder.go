package kernel

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

func BuildFactory(ctx context.Context, config cfg.Config, logger log.Logger, options []Option) (*factory, error) {
	blueprint := NewBlueprint(options...)

	var err error
	var factory *factory

	if factory, err = NewFactory(ctx, config, logger, blueprint); err != nil {
		return nil, err
	}

	return factory, nil
}

func BuildKernel(ctx context.Context, config cfg.Config, logger log.Logger, options []Option) (Kernel, error) {
	var err error
	var factory *factory

	if factory, err = BuildFactory(ctx, config, logger, options); err != nil {
		return nil, err
	}

	return factory.GetKernel(), nil
}
