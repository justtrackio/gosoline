package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/mon"
)

type ddbRepoFactory func() ddb.Repository

type dynamoDbFixtureWriter struct {
	logger  mon.Logger
	factory ddbRepoFactory
	purger  *dynamodbPurger
}

func DynamoDbFixtureWriterFactory(settings *ddb.Settings) FixtureWriterFactory {
	return func(config cfg.Config, logger mon.Logger) FixtureWriter {
		settings := &ddb.Settings{
			ModelId:    settings.ModelId,
			AutoCreate: true,
			Main: ddb.MainSettings{
				Model:              settings.Main.Model,
				ReadCapacityUnits:  1,
				WriteCapacityUnits: 1,
			},
			Global: settings.Global,
		}

		factory := func() ddb.Repository {
			return ddb.NewRepository(config, logger, settings)
		}

		purger := newDynamodbPurger(config, logger, settings)

		return NewDynamoDbFixtureWriterWithInterfaces(logger, factory, purger)
	}
}

func NewDynamoDbFixtureWriterWithInterfaces(logger mon.Logger, factory ddbRepoFactory, purger *dynamodbPurger) FixtureWriter {
	return &dynamoDbFixtureWriter{
		logger:  logger,
		factory: factory,
		purger:  purger,
	}
}

func (d *dynamoDbFixtureWriter) Purge() error {
	return d.purger.purgeDynamodb()
}

func (d *dynamoDbFixtureWriter) Write(fs *FixtureSet) error {
	repo := d.factory()

	for _, fixture := range fs.Fixtures {
		_, err := repo.PutItem(context.Background(), nil, fixture)

		if err != nil {
			return err
		}
	}

	d.logger.Infof("loaded %d dynamodb fixtures", len(fs.Fixtures))

	return nil
}
