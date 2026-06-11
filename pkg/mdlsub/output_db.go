package mdlsub

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	MaxPersistRetries = 2
	OutputTypeDb      = "db"
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
	orm, err := dbRepo.NewOrm(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	return NewOutputDbWithInterfaces(logger, orm), nil
}

func NewOutputDbWithInterfaces(logger log.Logger, orm *gorm.DB) *OutputDb {
	return &OutputDb{
		logger: logger,
		orm:    orm,
	}
}

func (p *OutputDb) Persist(ctx context.Context, model Model, op string) error {
	addressableModel, err := ensureAddressableModel(model)
	if err != nil {
		return err
	}

	switch op {
	case dbRepo.Create, dbRepo.Update:
		err = p.save(ctx, addressableModel)
	case dbRepo.Delete:
		err = p.orm.Delete(addressableModel).Error
	default:
		err = fmt.Errorf("unknown operation %s in OutputDb", op)
	}

	return err
}

// save persists the model via gorm's Save. Save is not atomic (UPDATE, then
// SELECT+INSERT on 0 rows), so we retry on duplicate-entry errors to take the UPDATE path.
func (p *OutputDb) save(ctx context.Context, model any) error {
	var err error

	for attempt := 0; attempt <= MaxPersistRetries; attempt++ {
		if err = p.orm.Save(model).Error; !db.IsDuplicateEntryError(err) {
			return err
		}

		p.logger.Warn(ctx, "retrying persist after duplicate entry error (attempt %d/%d): %w", attempt+1, MaxPersistRetries, err)
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
