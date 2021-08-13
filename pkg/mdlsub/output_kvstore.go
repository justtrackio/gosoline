package mdlsub

import (
	"context"
	"fmt"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/log"
)

const (
	OutputTypeKvstore = "kvstore"
)

func init() {
	outputFactories[OutputTypeKvstore] = outputKvstoreFactory
}

func outputKvstoreFactory(ctx context.Context, config cfg.Config, logger log.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) (map[int]Output, error) {
	var err error
	outputs := make(map[int]Output)

	for version := range transformers {
		if outputs[version], err = NewOutputKvstore(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("can not create output: %w", err)
		}
	}

	return outputs, nil
}

type OutputKvstore struct {
	logger log.Logger
	store  kvstore.KvStore
}

func NewOutputKvstore(ctx context.Context, config cfg.Config, logger log.Logger, settings *SubscriberSettings) (*OutputKvstore, error) {
	store, err := kvstore.NewConfigurableKvStore(ctx, config, logger, settings.TargetModel.Name)
	if err != nil {
		return nil, fmt.Errorf("can not create kvStore: %w", err)
	}

	return &OutputKvstore{
		logger: logger,
		store:  store,
	}, nil
}

func (p *OutputKvstore) Persist(ctx context.Context, model Model, op string) error {
	err := p.store.Put(ctx, model.GetId(), model)

	return err
}
