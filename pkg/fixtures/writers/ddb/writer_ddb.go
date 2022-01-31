package ddb

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/fixtures/writers"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ddbRepoFactory func() (ddb.Repository, error)

type dynamoDbFixtureWriter struct {
	logger  log.Logger
	factory ddbRepoFactory
	purger  writers.Purger
}

func DynamoDbFixtureWriterFactory(settings *ddb.Settings, options ...DdbWriterOption) writers.FixtureWriterFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (writers.FixtureWriter, error) {
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
		var purger writers.Purger

		if purger, err = newDynamodbPurger(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("can not create dynamodb purger: %w", err)
		}

		return NewDynamoDbFixtureWriterWithInterfaces(logger, factory, purger), nil
	}
}

func NewDynamoDbFixtureWriterWithInterfaces(logger log.Logger, factory ddbRepoFactory, purger writers.Purger) writers.FixtureWriter {
	return &dynamoDbFixtureWriter{
		logger:  logger,
		factory: factory,
		purger:  purger,
	}
}

func (d *dynamoDbFixtureWriter) Purge(ctx context.Context) error {
	return d.purger.Purge(ctx)
}

func (d *dynamoDbFixtureWriter) Write(ctx context.Context, fs *writers.FixtureSet) error {
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
