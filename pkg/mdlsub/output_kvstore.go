package mdlsub

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mon"
)

func init() {
	outputFactories["kvstore"] = outputKvstoreFactory
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
	store := kvstore.NewChainKvStore(config, logger, false, &kvstore.Settings{
		AppId: cfg.AppId{
			Project:     settings.TargetModel.Project,
			Family:      settings.TargetModel.Family,
			Application: settings.TargetModel.Application,
		},
		Name: settings.TargetModel.Name,
	})
	store.Add(kvstore.NewRedisKvStore)
	store.Add(kvstore.NewDdbKvStore)

	return &OutputKvstore{
		logger: logger,
		store:  store,
	}
}

func (p *OutputKvstore) Persist(ctx context.Context, model Model, op string) error {
	err := p.store.Put(ctx, model.GetId(), model)

	return err
}
