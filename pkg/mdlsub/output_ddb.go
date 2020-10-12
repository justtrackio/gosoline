package mdlsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mon"
)

func init() {
	outputFactories["ddb"] = outputDdbFactory
}

func repoInit(config cfg.Config, logger mon.Logger, settings *SubscriberSettings) func(model interface{}) ddb.Repository {
	return func(model interface{}) ddb.Repository {
		repo := ddb.NewRepository(config, logger, &ddb.Settings{
			ModelId: settings.TargetModel,
			Main: ddb.MainSettings{
				Model:              model,
				ReadCapacityUnits:  5,
				WriteCapacityUnits: 5,
			},
		})

		return ddb.NewMetricRepository(config, logger, repo)
	}
}

func outputDdbFactory(config cfg.Config, logger mon.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) map[int]Output {
	outputs := make(map[int]Output)

	for version := range transformers {
		outputs[version] = NewOutputDdb(config, logger, settings)
	}

	return outputs
}

type OutputDdb struct {
	repoInit func(model interface{}) ddb.Repository
	repo     ddb.Repository
}

func NewOutputDdb(config cfg.Config, logger mon.Logger, settings *SubscriberSettings) *OutputDdb {
	return &OutputDdb{
		repoInit: repoInit(config, logger, settings),
	}
}

func (p *OutputDdb) GetType() string {
	return "ddb"
}

func (p *OutputDdb) Persist(ctx context.Context, model Model, op string) error {
	if p.repo == nil {
		p.repo = p.repoInit(model)
	}

	var err error

	switch op {
	case ddb.Create, ddb.Update:
		_, err = p.repo.PutItem(ctx, nil, model)
	case ddb.Delete:
		_, err = p.repo.DeleteItem(ctx, nil, model)
	default:
		err = fmt.Errorf("unknown operation %s in OutputDdb", op)
	}

	return err
}
