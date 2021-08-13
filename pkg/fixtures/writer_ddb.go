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
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureWriter, error) {
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
			return ddb.NewRepository(ctx, config, logger, settings)
		}

		var err error
		var purger *dynamodbPurger

		if purger, err = newDynamodbPurger(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("can not create dynamodb purger: %w", err)
		}

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

func (d *dynamoDbFixtureWriter) Purge(ctx context.Context) error {
	return d.purger.purgeDynamodb(ctx)
}

func (d *dynamoDbFixtureWriter) Write(ctx context.Context, fs *FixtureSet) error {
	if len(fs.Fixtures) == 0 {
		return nil
	}

	repo, err := d.factory()
	if err != nil {
		return fmt.Errorf("can not create ddb repository: %w", err)
	}

	if _, err = repo.BatchPutItems(ctx, fs.Fixtures); err != nil {
		return err
	}

	d.logger.Info("loaded %d dynamodb fixtures", len(fs.Fixtures))

	return nil
}
