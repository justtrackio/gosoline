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
	config cfg.Config
	logger mon.Logger
}

func NewMySqlFixtureWriter(config cfg.Config, logger mon.Logger) FixtureWriter {
	return cachedWriters.New("mysql", func() FixtureWriter {
		return &mySqlFixtureWriter{
			config: config,
			logger: logger,
		}
	})
}

func (m *mySqlFixtureWriter) WriteFixtures(fs *FixtureSet) error {
	r, err := m.GetRepository(fs)

	if err != nil {
		return err
	}

	if r == nil {
		metaData := fs.WriterMetadata.(db_repo.Metadata)
		return fmt.Errorf("could not create repository for for model %s", metaData.ModelId.String())
	}

	ctx := context.Background()

	for _, item := range fs.Fixtures {
		model, ok := item.(db_repo.ModelBased)

		if !ok {
			return fmt.Errorf("invalid fixture type: %s", reflect.TypeOf(item))
		}

		err := r.Create(ctx, model)

		if err != nil {
			return err
		}
	}

	m.logger.Infof("loaded %d mysql fixtures", len(fs.Fixtures))

	return nil
}

func (m *mySqlFixtureWriter) GetRepository(fs *FixtureSet) (db_repo.Repository, error) {
	metadata, ok := fs.WriterMetadata.(db_repo.Metadata)

	if !ok {
		return nil, fmt.Errorf("invalid writer metadata type: %s", reflect.TypeOf(fs.WriterMetadata))
	}

	metadata.ModelId.PadFromConfig(m.config)

	settings := db_repo.Settings{
		AppId:    cfg.GetAppIdFromConfig(m.config),
		Metadata: metadata,
	}

	return db_repo.New(m.config, m.logger, settings), nil
}
