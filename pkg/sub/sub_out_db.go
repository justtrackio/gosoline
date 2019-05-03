package sub

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/jinzhu/gorm"
)

type subOutDb struct {
	logger mon.Logger
	orm    *gorm.DB
}

func (p *subOutDb) Boot(config cfg.Config, logger mon.Logger, settings Settings) error {
	p.logger = logger
	p.orm = db_repo.NewOrm(config, logger)

	return nil
}

func (p *subOutDb) Persist(ctx context.Context, model Model, op string) error {
	var err error

	switch op {
	case db_repo.Create:
		err = p.orm.Create(model).Error
	case db_repo.Update:
		err = p.orm.Save(model).Error
	case db_repo.Delete:
		err = p.orm.Delete(model).Error
	default:
		err = fmt.Errorf("unknown operation %s in subOutDb", op)
	}

	return err
}
