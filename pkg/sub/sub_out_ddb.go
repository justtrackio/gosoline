package sub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mon"
)

func repoInit(config cfg.Config, logger mon.Logger, settings Settings) func(model interface{}) ddb.Repository {
	return func(model interface{}) ddb.Repository {
		repo := ddb.NewRepository(config, logger, &ddb.Settings{
			ModelId: settings.TargetModelId,
			Main: ddb.MainSettings{
				Model:              model,
				ReadCapacityUnits:  5,
				WriteCapacityUnits: 5,
			},
			Backoff: settings.Backoff,
		})

		return ddb.NewMetricRepository(config, logger, repo)
	}
}

type subOutDdb struct {
	repoInit func(model interface{}) ddb.Repository
	repo     ddb.Repository
}

func (p *subOutDdb) GetType() string {
	return "ddb"
}

func (p *subOutDdb) Boot(config cfg.Config, logger mon.Logger, settings Settings) error {
	p.repoInit = repoInit(config, logger, settings)

	return nil
}

func (p *subOutDdb) Persist(ctx context.Context, model Model, op string) error {
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
		err = fmt.Errorf("unknown operation %s in subOutDdb", op)
	}

	return err
}
