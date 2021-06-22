package fixtures

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/log"
)

type ddbRepoFactory func() (ddb.Repository, error)

type dynamoDbFixtureWriter struct {
	logger  log.Logger
	factory ddbRepoFactory
	purger  *dynamodbPurger
}

func DynamoDbFixtureWriterFactory(settings *ddb.Settings, options ...DdbWriterOption) FixtureWriterFactory {
	return func(config cfg.Config, logger log.Logger) (FixtureWriter, error) {
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

		for _, opt := range options {
			opt(settings)
		}

		factory := func() (ddb.Repository, error) {
			return ddb.NewRepository(config, logger, settings)
		}

		purger := newDynamodbPurger(config, logger, settings)

		return NewDynamoDbFixtureWriterWithInterfaces(logger, factory, purger), nil
	}
}

func NewDynamoDbFixtureWriterWithInterfaces(logger log.Logger, factory ddbRepoFactory, purger *dynamodbPurger) FixtureWriter {
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
	if len(fs.Fixtures) == 0 {
		return nil
	}

	repo, err := d.factory()
	if err != nil {
		return fmt.Errorf("can not create ddb repository: %w", err)
	}

	if _, err = repo.BatchPutItems(context.Background(), fs.Fixtures); err != nil {
		return err
	}

	d.logger.Info("loaded %d dynamodb fixtures", len(fs.Fixtures))

	return nil
}
