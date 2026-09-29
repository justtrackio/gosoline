package mdlsub

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	MaxPersistRetries = 2
	OutputTypeDb      = "db"
)

func init() {
	AddOutput(OutputTypeDb, outputDbFactory)
}

func outputDbFactory(
	ctx context.Context,
	config cfg.Config,
	logger log.Logger,
	_ *SubscriberSettings,
	transformers VersionedModelTransformers,
	_ string,
) (map[int]Output, error) {
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

type outputDbOrmSettings struct {
	Driver      string `cfg:"driver" validation:"required"`
	Application string `cfg:"application" default:"{app.name}"`
	Migrations  struct {
		TablePrefixed bool `cfg:"table_prefixed" default:"true"`
	} `cfg:"migrations"`
}

type outputDbOrmClient struct {
	db.Client
}

func (c outputDbOrmClient) Exec(query string, args ...any) (sql.Result, error) {
	return c.Client.Exec(context.Background(), query, args...)
}

func (c outputDbOrmClient) Prepare(query string) (*sql.Stmt, error) {
	return c.Client.Prepare(context.Background(), query)
}

func (c outputDbOrmClient) Query(query string, args ...any) (*sql.Rows, error) {
	return c.Client.Query(context.Background(), query, args...)
}

func (c outputDbOrmClient) QueryRow(query string, args ...any) *sql.Row {
	return c.Client.QueryRow(context.Background(), query, args...)
}

type outputDbNoopLogger struct{}

func (outputDbNoopLogger) Print(...any) {}

func NewOutputDb(ctx context.Context, config cfg.Config, logger log.Logger) (*OutputDb, error) {
	client, err := db.NewClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create orm: can not create db connection: %w", err)
	}

	var settings outputDbOrmSettings
	if err := config.UnmarshalKey("db.default", &settings); err != nil {
		return nil, fmt.Errorf("can not create orm: failed to unmarshal orm settings for key %q: %w", "db.default", err)
	}

	orm, err := gorm.Open(settings.Driver, outputDbOrmClient{client})
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	orm.LogMode(false)
	orm.SetLogger(outputDbNoopLogger{})
	orm = orm.Set("gorm:auto_preload", true)
	orm = orm.Set("gorm:save_associations", false)
	orm.SetNowFuncOverride(func() time.Time {
		return clock.Provider.Now()
	})

	if settings.Migrations.TablePrefixed {
		prefix := strcase.ToSnake(settings.Application)
		gorm.DefaultTableNameHandler = func(_ *gorm.DB, table string) string {
			return fmt.Sprintf("%s_%s", prefix, table)
		}
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
	case TypeCreate, TypeUpdate:
		err = p.save(ctx, addressableModel)
	case TypeDelete:
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
