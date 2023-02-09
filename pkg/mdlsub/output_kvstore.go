package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
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
	store  kvstore.KvStore[Model]
}

func NewOutputKvstore(ctx context.Context, config cfg.Config, logger log.Logger, settings *SubscriberSettings) (*OutputKvstore, error) {
	store, err := kvstore.NewConfigurableKvStore[Model](ctx, config, logger, settings.TargetModel.Name)
	if err != nil {
		return nil, fmt.Errorf("can not create kvStore: %w", err)
	}

	return &OutputKvstore{
		logger: logger,
		store:  store,
	}, nil
}

func (p *OutputKvstore) Persist(ctx context.Context, model Model, op string) error {
	var err error

	switch op {
	case TypeCreate, TypeUpdate:
		err = p.store.Put(ctx, model.GetId(), model)
	case TypeDelete:
		err = p.store.Delete(ctx, model.GetId())
	default:
		err = fmt.Errorf("unknown operation %s in OutputKvStore", op)
	}

	return err
}
