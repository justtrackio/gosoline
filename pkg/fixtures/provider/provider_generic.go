package provider

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Data[M any, D any] struct {
	Metadata M `json:"metadata"`
	Data     D `json:"data"`
}

type fixturesProvider[M any, D any, E any] struct {
	exporterFactory func(clientName string) (E, error)
	metadata        *appctx.Metadata
	metadataKey     string
	getName         func(M) string
	exportFunc      func(ctx context.Context, exporter E, metadata M) (D, error)
}

func NewFixtureProviderFactory[M any, D any, E any](
	metadataKey string,
	provideExporterFunc func(ctx context.Context, config cfg.Config, logger log.Logger, clientName string) (E, error),
	getName func(m M) string,
	exportFunc func(ctx context.Context, exporter E, metadata M) (D, error),
) FixturesProviderFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger, metadata *appctx.Metadata) (FixturesProvider, error) {
		exporterFactory := func(clientName string) (E, error) {
			return provideExporterFunc(ctx, config, logger, clientName)
		}

		return &fixturesProvider[M, D, E]{
			exporterFactory: exporterFactory,
			metadata:        metadata,
			metadataKey:     metadataKey,
			getName:         getName,
			exportFunc:      exportFunc,
		}, nil
	}
}

func (d *fixturesProvider[M, D, E]) Provide(ctx context.Context) (any, error) {
	metadataS, err := d.metadata.Get(d.metadataKey).Slice()
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for key %s: %w", d.metadataKey, err)
	}

	res := make([]Data[M, D], len(metadataS))

	var m M
	var ok bool
	var data D
	var exporter E

	for i, metadataItem := range metadataS {
		if m, ok = metadataItem.(M); !ok {
			return nil, fmt.Errorf("metadata format incorrect, expected %T, got %T", *new(M), metadataItem)
		}

		name := d.getName(m)
		exporter, err = d.exporterFactory(name)
		if err != nil {
			return nil, fmt.Errorf("failed to create exporter for name %s: %w", name, err)
		}

		data, err = d.exportFunc(ctx, exporter, m)
		if err != nil {
			return nil, err
		}

		res[i] = Data[M, D]{
			Metadata: m,
			Data:     data,
		}
	}

	return res, nil
}
