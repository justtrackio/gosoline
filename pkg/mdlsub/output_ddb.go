package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	OutputTypeDdb = "ddb"
)

func init() {
	outputFactories[OutputTypeDdb] = outputDdbFactory
}

func repoInit(ctx context.Context, config cfg.Config, logger log.Logger, settings *SubscriberSettings) func(model any) (ddb.Repository, error) {
	return func(model any) (ddb.Repository, error) {
		repo, err := ddb.NewRepository(ctx, config, logger, &ddb.Settings{
			ModelId: settings.TargetModel.ModelId,
			Main: ddb.MainSettings{
				Model: model,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("can not create ddb repository: %w", err)
		}

		return ddb.NewMetricRepository(config, logger, repo), nil
	}
}

func outputDdbFactory(ctx context.Context, config cfg.Config, logger log.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) (map[int]Output, error) {
	outputs := make(map[int]Output)

	for version := range transformers {
		outputs[version] = NewOutputDdb(ctx, config, logger, settings)
	}

	return outputs, nil
}

type OutputDdb struct {
	repo conc.Lazy[ddb.Repository, any]
}

func NewOutputDdb(ctx context.Context, config cfg.Config, logger log.Logger, settings *SubscriberSettings) *OutputDdb {
	return &OutputDdb{
		repo: conc.NewLazy(repoInit(ctx, config, logger, settings)),
	}
}

func (p *OutputDdb) GetType() string {
	return "ddb"
}

func (p *OutputDdb) Persist(ctx context.Context, model Model, op string) error {
	var err error
	var repo ddb.Repository

	if repo, err = p.repo.Get(model); err != nil {
		return fmt.Errorf("can not initialize ddb repository: %w", err)
	}

	switch op {
	case ddb.Create, ddb.Update:
		_, err = repo.PutItem(ctx, nil, model)
	case ddb.Delete:
		_, err = repo.DeleteItem(ctx, nil, model)
	default:
		err = fmt.Errorf("unknown operation %s in OutputDdb", op)
	}

	return err
}
