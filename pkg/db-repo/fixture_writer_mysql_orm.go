package db_repo

import (
	"context"
	"fmt"
	"reflect"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

type mysqlOrmFixtureWriter struct {
	logger   log.Logger
	metadata *Metadata
	repo     ConfigurableRepository[uint, ModelBased[uint]]
}

func MysqlOrmFixtureSetFactory[T ModelBased[uint]](metadata *Metadata, data fixtures.NamedFixtures[T], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewMysqlOrmFixtureWriter(ctx, config, logger, metadata); err != nil {
			return nil, fmt.Errorf("failed to create mysql orm fixture writer for %s: %w", metadata.ModelId.String(), err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewMysqlOrmFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, metadata *Metadata) (fixtures.FixtureWriter, error) {
	if err := metadata.ModelId.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("can not pad model id from config: %w", err)
	}

	appId, err := cfg.GetAppIdFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("can not get app id from config: %w", err)
	}

	repoSettings := Settings{
		AppId:      appId,
		Metadata:   *metadata,
		ClientName: "default",
	}

	var dbSettings *db.Settings
	var repo ConfigurableRepository[uint, ModelBased[uint]]

	if dbSettings, err = db.ReadSettings(config, "default"); err != nil {
		return nil, fmt.Errorf("can not create repo: %w", err)
	}
	dbSettings.Parameters["FOREIGN_KEY_CHECKS"] = "0"

	if repo, err = NewWithDbSettings[uint, ModelBased[uint]](ctx, config, logger, dbSettings, repoSettings); err != nil {
		return nil, fmt.Errorf("can not create repo: %w", err)
	}

	return NewMysqlFixtureWriterWithInterfaces(logger, metadata, repo), nil
}

func NewMysqlFixtureWriterWithInterfaces(logger log.Logger, metadata *Metadata, repo ConfigurableRepository[uint, ModelBased[uint]]) fixtures.FixtureWriter {
	return &mysqlOrmFixtureWriter{
		logger:   logger,
		metadata: metadata,
		repo:     repo,
	}
}

func (m *mysqlOrmFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	var ok bool
	var model ModelBased[uint]

	for _, item := range fixtures {
		if model, ok = item.(ModelBased[uint]); !ok {
			return fmt.Errorf("assertion failed: %T is not ModelBased[uint]", item)
		}

		m.repo.SetModelSource(func() ModelBased[uint] {
			return createFromType(model)
		})

		err := m.repo.Update(ctx, model)
		if err != nil {
			return err
		}
	}

	m.logger.Info(ctx, "loaded %d mysql fixtures", len(fixtures))

	return nil
}

func createFromType[T any](model T) T {
	modelType := reflect.TypeOf(model)

	switch modelType.Kind() {
	case reflect.Pointer:
		return reflect.New(modelType.Elem()).Interface().(T)

	case reflect.Map:
		return reflect.MakeMap(modelType).Interface().(T)

	default:
		return *reflect.New(modelType.Elem()).Interface().(*T)
	}
}
