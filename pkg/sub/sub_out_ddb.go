package sub

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mon"
)

type subOutDdb struct {
	logger mon.Logger
	repo   ddb.Repository
}

func (p *subOutDdb) GetType() string {
	return "ddb"
}

func (p *subOutDdb) Boot(config cfg.Config, logger mon.Logger, settings Settings) error {
	p.logger = logger

	repo := ddb.New(config, logger, ddb.Settings{
		ModelId:            settings.TargetModelId,
		ReadCapacityUnits:  5,
		WriteCapacityUnits: 5,
	})
	p.repo = ddb.NewMetricRepository(config, logger, repo)

	return nil
}

func (p *subOutDdb) Persist(ctx context.Context, model Model, op string) error {
	logger := p.logger.WithContext(ctx)
	err := p.repo.CreateTable(model)

	if err != nil {
		logger.Error(err, "could not create ddb table for model")
		return err
	}

	err = p.repo.Save(ctx, model)

	return err
}
