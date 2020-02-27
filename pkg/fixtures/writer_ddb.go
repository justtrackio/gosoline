package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mon"
)

type dynamoDbFixtureWriter struct {
	config   cfg.Config
	logger   mon.Logger
	settings *ddb.Settings
}

func DynamoDbFixtureWriterFactory(settings *ddb.Settings) FixtureWriterFactory {
	return func(cfg cfg.Config, logger mon.Logger) FixtureWriter {
		writer := &dynamoDbFixtureWriter{
			config:   cfg,
			logger:   logger,
			settings: settings,
		}

		return writer
	}
}

func (d *dynamoDbFixtureWriter) Write(fs *FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		d.logger.Info("loaded 0 dynamo db fixtures")
		return nil
	}

	repo := ddb.NewRepository(d.config, d.logger, &ddb.Settings{
		ModelId:    d.settings.ModelId,
		AutoCreate: true,
		Main: ddb.MainSettings{
			Model:              d.settings.Main.Model,
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
		Global: d.settings.Global,
	})

	for _, fixture := range fs.Fixtures {
		_, err := repo.PutItem(context.Background(), nil, fixture)

		if err != nil {
			return err
		}
	}

	d.logger.Infof("loaded %d dynamo db fixtures", len(fs.Fixtures))

	return nil
}
