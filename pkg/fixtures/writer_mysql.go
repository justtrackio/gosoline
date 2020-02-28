package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	db_repo "github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/mon"
	"reflect"
)

type mysqlFixtureWriter struct {
	logger mon.Logger
	repo   db_repo.Repository
}

func MysqlFixtureWriterFactory(metadata *db_repo.Metadata) FixtureWriterFactory {
	return func(config cfg.Config, logger mon.Logger) FixtureWriter {
		metadata.ModelId.PadFromConfig(config)

		settings := db_repo.Settings{
			AppId:    cfg.GetAppIdFromConfig(config),
			Metadata: *metadata,
		}

		repo := db_repo.New(config, logger, settings)

		return NewMysqlFixtureWriterWithInterfaces(logger, repo)
	}
}

func NewMysqlFixtureWriterWithInterfaces(logger mon.Logger, repo db_repo.Repository) FixtureWriter {
	return &mysqlFixtureWriter{
		logger: logger,
		repo:   repo,
	}
}

func (m *mysqlFixtureWriter) Write(fs *FixtureSet) error {
	ctx := context.Background()

	for _, item := range fs.Fixtures {
		model, ok := item.(db_repo.ModelBased)

		if !ok {
			return fmt.Errorf("invalid fixture type: %s", reflect.TypeOf(item))
		}

		err := m.repo.Update(ctx, model)

		if err != nil {
			return err
		}
	}

	m.logger.Infof("loaded %d mysql fixtures", len(fs.Fixtures))

	return nil
}
