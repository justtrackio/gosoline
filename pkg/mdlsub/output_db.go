package mdlsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/log"
	"github.com/jinzhu/gorm"
)

const (
	OutputTypeDb = "db"
)

func init() {
	outputFactories[OutputTypeDb] = outputDbFactory
}

func outputDbFactory(config cfg.Config, logger log.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) (map[int]Output, error) {
	var err error
	var outputs = make(map[int]Output)

	for version := range transformers {
		if outputs[version], err = NewOutputDb(config, logger); err != nil {
			return nil, fmt.Errorf("can not create outputDb: %w", err)
		}
	}

	return outputs, nil
}

type OutputDb struct {
	logger log.Logger
	orm    *gorm.DB
}

func NewOutputDb(config cfg.Config, logger log.Logger) (*OutputDb, error) {
	orm, err := db_repo.NewOrm(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	return &OutputDb{
		logger: logger,
		orm:    orm,
	}, nil
}

func (p *OutputDb) Persist(ctx context.Context, model Model, op string) error {
	var err error

	switch op {
	case db_repo.Create:
		err = p.orm.Create(model).Error
	case db_repo.Update:
		err = p.orm.Save(model).Error
	case db_repo.Delete:
		err = p.orm.Delete(model).Error
	default:
		err = fmt.Errorf("unknown operation %s in OutputDb", op)
	}

	return err
}
