package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ddbRepoFactory func() (ddb.Repository, error)

type dynamoDbFixtureWriter struct {
	logger  log.Logger
	factory ddbRepoFactory
	purger  *dynamodbPurger
}

func NewDynamoDbFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, settings *ddb.Settings, options ...DdbWriterOption) (FixtureWriter, error) {
	ddbSettings := &ddb.Settings{
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
		opt(ddbSettings)
	}

	factory := func() (ddb.Repository, error) {
		return ddb.NewRepository(ctx, config, logger, ddbSettings)
	}

	var err error
	var purger *dynamodbPurger

	if purger, err = newDynamodbPurger(ctx, config, logger, ddbSettings); err != nil {
		return nil, fmt.Errorf("can not create dynamodb purger: %w", err)
	}

	return NewDynamoDbFixtureWriterWithInterfaces(logger, factory, purger), nil
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

func (d *dynamoDbFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	if len(fixtures) == 0 {
		return nil
	}

	repo, err := d.factory()
	if err != nil {
		return fmt.Errorf("can not create ddb repository: %w", err)
	}

	if _, err = repo.BatchPutItems(ctx, fixtures); err != nil {
		return err
	}

	d.logger.Info("loaded %d dynamodb fixtures", len(fixtures))

	return nil
}
