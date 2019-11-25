package sub

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mon"
)

type subOutKvstore struct {
	logger mon.Logger
	store  kvstore.KvStore
}

func (p *subOutKvstore) GetType() string {
	return "kvstore"
}

func (p *subOutKvstore) Boot(config cfg.Config, logger mon.Logger, settings Settings) error {
	p.logger = logger

	store := kvstore.NewChainKvStore(config, logger, &kvstore.Settings{
		AppId: cfg.AppId{
			Application: config.GetString("app_group"),
		},
		Name: settings.TargetModelId.Name,
	})
	store.Add(kvstore.NewRedisKvStore)
	store.Add(kvstore.NewDdbKvStore)

	p.store = store

	return nil
}

func (p *subOutKvstore) Persist(ctx context.Context, model Model, op string) error {
	err := p.store.Put(ctx, model.GetId(), model)

	return err
}
