package mdlsub

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	OutputTypeDb = "db"
)

func init() {
	outputFactories[OutputTypeDb] = outputDbFactory
}

func outputDbFactory(ctx context.Context, config cfg.Config, logger log.Logger, _ *SubscriberSettings, transformers VersionedModelTransformers) (map[int]Output, error) {
	var err error
	outputs := make(map[int]Output)

	for version := range transformers {
		if outputs[version], err = NewOutputDb(ctx, config, logger); err != nil {
			return nil, fmt.Errorf("can not create outputDb: %w", err)
		}
	}

	return outputs, nil
}

type OutputDb struct {
	logger log.Logger
	orm    *gorm.DB
}

func NewOutputDb(ctx context.Context, config cfg.Config, logger log.Logger) (*OutputDb, error) {
	orm, err := db_repo.NewOrm(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	return &OutputDb{
		logger: logger,
		orm:    orm,
	}, nil
}

func (p *OutputDb) Persist(_ context.Context, model Model, op string) error {
	addressableModel, err := ensureAddressableModel(model)
	if err != nil {
		return err
	}

	switch op {
	case db_repo.Create, db_repo.Update:
		err = p.orm.Save(addressableModel).Error
	case db_repo.Delete:
		err = p.orm.Delete(addressableModel).Error
	default:
		err = fmt.Errorf("unknown operation %s in OutputDb", op)
	}

	return err
}

func ensureAddressableModel(model Model) (any, error) {
	value := reflect.ValueOf(model)
	if !value.IsValid() {
		return nil, fmt.Errorf("model must not be nil")
	}
	if value.Kind() == reflect.Ptr {
		return nil, fmt.Errorf("model must not be a pointer")
	}

	pointer := reflect.New(value.Type())
	pointer.Elem().Set(value)

	return pointer.Interface(), nil
}
