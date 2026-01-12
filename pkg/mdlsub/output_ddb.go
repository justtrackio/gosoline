package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	OutputTypeDdb = "ddb"
)

func init() {
	outputFactories[OutputTypeDdb] = outputDdbFactory
}

func outputDdbFactory(ctx context.Context, config cfg.Config, logger log.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) (map[int]Output, error) {
	outputs := make(map[int]Output)

	for version, transformer := range transformers {
		model, err := transformer.GetModel()
		if err != nil {
			return nil, fmt.Errorf("can not get model from transformer: %w", err)
		}

		var output Output
		if output, err = NewOutputDdb(ctx, config, logger, model, settings); err != nil {
			return nil, fmt.Errorf("can not create ddb output: %w", err)
		}

		outputs[version] = output
	}

	return outputs, nil
}

type OutputDdb struct {
	repo ddb.Repository
}

func NewOutputDdb(ctx context.Context, config cfg.Config, logger log.Logger, model any, settings *SubscriberSettings) (*OutputDdb, error) {
	var err error
	var repo ddb.Repository

	ddbSettings := &ddb.Settings{
		ModelId: settings.TargetModel.ModelId,
		Main: ddb.MainSettings{
			Model: model,
		},
	}

	if repo, err = ddb.NewRepository(ctx, config, logger, ddbSettings); err != nil {
		return nil, fmt.Errorf("can not create ddb repository: %w", err)
	}

	return &OutputDdb{
		repo: ddb.NewMetricRepository(config, logger, repo),
	}, nil
}

func (p *OutputDdb) GetType() string {
	return "ddb"
}

func (p *OutputDdb) Persist(ctx context.Context, model Model, op string) error {
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
