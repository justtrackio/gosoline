package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	db_repo "github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/mon"
	"reflect"
)

type mySqlFixtureWriter struct {
	config   cfg.Config
	logger   mon.Logger
	metadata *db_repo.Metadata
}

func MySqlFixtureWriterFactory(metadata *db_repo.Metadata) FixtureWriterFactory {
	return func(cfg cfg.Config, logger mon.Logger) FixtureWriter {
		writer := newMySqlFixtureWriter(cfg, logger)
		writer.WithMetadata(metadata)

		return writer
	}
}

func newMySqlFixtureWriter(config cfg.Config, logger mon.Logger) *mySqlFixtureWriter {
	return &mySqlFixtureWriter{
		config: config,
		logger: logger,
	}
}

func (m *mySqlFixtureWriter) WithMetadata(metadata *db_repo.Metadata) {
	m.metadata = metadata
}

func (m *mySqlFixtureWriter) WriteFixtures(fs *FixtureSet) error {
	r, err := m.GetRepository(fs)

	if err != nil {
		return err
	}

	if r == nil {
		return fmt.Errorf("could not create repository for for model %s", m.metadata.ModelId.String())
	}

	ctx := context.Background()

	for _, item := range fs.Fixtures {
		model, ok := item.(db_repo.ModelBased)

		if !ok {
			return fmt.Errorf("invalid fixture type: %s", reflect.TypeOf(item))
		}

		err := r.Update(ctx, model)

		if err != nil {
			return err
		}
	}

	m.logger.Infof("loaded %d mysql fixtures", len(fs.Fixtures))

	return nil
}

func (m *mySqlFixtureWriter) GetRepository(fs *FixtureSet) (db_repo.Repository, error) {
	m.metadata.ModelId.PadFromConfig(m.config)

	settings := db_repo.Settings{
		AppId:    cfg.GetAppIdFromConfig(m.config),
		Metadata: *m.metadata,
	}

	return db_repo.New(m.config, m.logger, settings), nil
}
