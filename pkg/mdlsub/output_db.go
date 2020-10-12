package mdlsub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jinzhu/gorm"
)

func init() {
	outputFactories["db"] = outputDbFactory
}

func outputDbFactory(config cfg.Config, logger mon.Logger, settings *SubscriberSettings, transformers VersionedModelTransformers) map[int]Output {
	outputs := make(map[int]Output)

	for version := range transformers {
		outputs[version] = NewOutputDb(config, logger)
	}

	return outputs
}

type OutputDb struct {
	logger mon.Logger
	orm    *gorm.DB
}

func NewOutputDb(config cfg.Config, logger mon.Logger) *OutputDb {
	orm := db_repo.NewOrm(config, logger)

	return &OutputDb{
		logger: logger,
		orm:    orm,
	}
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
