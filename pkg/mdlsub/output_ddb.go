package mdlsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/log"
)

const (
	OutputTypeDdb = "ddb"
)

func init() {
	outputFactories[OutputTypeDdb] = outputDdbFactory
}

func repoInit(config cfg.Config, logger log.Logger, settings *SubscriberSettings) func(model interface{}) (ddb.Repository, error) {
	return func(model interface{}) (ddb.Repository, error) {
		repo, err := ddb.NewRepository(config, logger, &ddb.Settings{
			ModelId: settings.TargetModel,
			Main: ddb.MainSettings{
				Model:              model,
				ReadCapacityUnits:  5,
				WriteCapacityUnits: 5,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("can not create ddb repository: %w", err)
		}

		return ddb.NewMetricRepository(config, logger, repo), nil
	}
}

func outputDdbFactory(config cfg.Config, logger log.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) (map[int]Output, error) {
	outputs := make(map[int]Output)

	for version := range transformers {
		outputs[version] = NewOutputDdb(config, logger, settings)
	}

	return outputs, nil
}

type OutputDdb struct {
	repoInit func(model interface{}) (ddb.Repository, error)
	repo     ddb.Repository
}

func NewOutputDdb(config cfg.Config, logger log.Logger, settings *SubscriberSettings) *OutputDdb {
	return &OutputDdb{
		repoInit: repoInit(config, logger, settings),
	}
}

func (p *OutputDdb) GetType() string {
	return "ddb"
}

func (p *OutputDdb) Persist(ctx context.Context, model Model, op string) error {
	var err error

	if p.repo == nil {
		if p.repo, err = p.repoInit(model); err != nil {
			return fmt.Errorf("can not initialize ddb repository: %w", err)
		}
	}

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
