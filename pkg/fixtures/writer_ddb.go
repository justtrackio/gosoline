package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mon"
)

type dynamoDbFixtureWriter struct {
	logger mon.Logger
	repo   ddb.Repository
}

func DynamoDbFixtureWriterFactory(settings *ddb.Settings) FixtureWriterFactory {
	return func(config cfg.Config, logger mon.Logger) FixtureWriter {
		repo := ddb.NewRepository(config, logger, &ddb.Settings{
			ModelId:    settings.ModelId,
			AutoCreate: true,
			Main: ddb.MainSettings{
				Model:              settings.Main.Model,
				ReadCapacityUnits:  1,
				WriteCapacityUnits: 1,
			},
			Global: settings.Global,
		})

		return NewDynamoDbFixtureWriterWithInterfaces(logger, repo)
	}
}

func NewDynamoDbFixtureWriterWithInterfaces(logger mon.Logger, repo ddb.Repository) FixtureWriter {
	return &dynamoDbFixtureWriter{
		logger: logger,
		repo:   repo,
	}
}

func (d *dynamoDbFixtureWriter) Write(fs *FixtureSet) error {
	for _, fixture := range fs.Fixtures {
		_, err := d.repo.PutItem(context.Background(), nil, fixture)

		if err != nil {
			return err
		}
	}

	d.logger.Infof("loaded %d dynamo db fixtures", len(fs.Fixtures))

	return nil
}
