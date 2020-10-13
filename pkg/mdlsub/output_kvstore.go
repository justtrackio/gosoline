package mdlsub

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mon"
)

const (
	OutputTypeKvstore = "kvstore"
)

func init() {
	outputFactories[OutputTypeKvstore] = outputKvstoreFactory
}

func outputKvstoreFactory(config cfg.Config, logger mon.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) map[int]Output {
	outputs := make(map[int]Output)

	for version := range transformers {
		outputs[version] = NewOutputKvstore(config, logger, settings)
	}

	return outputs
}

type OutputKvstore struct {
	logger mon.Logger
	store  kvstore.KvStore
}

func NewOutputKvstore(config cfg.Config, logger mon.Logger, settings *SubscriberSettings) *OutputKvstore {
	store := kvstore.NewConfigurableKvStore(config, logger, settings.TargetModel.Name)

	return &OutputKvstore{
		logger: logger,
		store:  store,
	}
}

func (p *OutputKvstore) Persist(ctx context.Context, model Model, op string) error {
	err := p.store.Put(ctx, model.GetId(), model)

	return err
}
