package mdlsub

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	db_repo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	OutputTypeDb = "db"
)

func init() {
	outputFactories[OutputTypeDb] = outputDbFactory
}

func outputDbFactory(_ context.Context, config cfg.Config, logger log.Logger, _ *SubscriberSettings, transformers VersionedModelTransformers) (map[int]Output, error) {
	var err error
	outputs := make(map[int]Output)

	for version := range transformers {
		if outputs[version], err = NewOutputDb(config, logger); err != nil {
			return nil, fmt.Errorf("can not create outputDb: %w", err)
		}
	}

	return outputs, nil
}

type OutputDb struct {
	logger log.Logger
	orm    db_repo.Remote
}

func NewOutputDb(config cfg.Config, logger log.Logger) (*OutputDb, error) {
	orm, err := db_repo.NewOrm(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	return NewOutputDbFactory(logger)(orm), nil
}

func NewOutputDbFactory(logger log.Logger) func(remote db_repo.Remote) *OutputDb {
	return func(remote db_repo.Remote) *OutputDb {
		return &OutputDb{
			logger: logger,
			orm:    remote,
		}
	}
}

func (p *OutputDb) Persist(_ context.Context, model Model, op string) error {
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
