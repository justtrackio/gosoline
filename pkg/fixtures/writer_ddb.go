package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
)

type dynamoDbFixtureWriter struct {
	logger log.Logger
	repo   ddb.Repository
	purger *dynamodbPurger
}

func DynamoDbFixtureSetFactory[T any](settings *ddb.Settings, data NamedFixtures[T], options ...FixtureSetOption) FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (FixtureSet, error) {
		var err error
		var writer FixtureWriter

		if writer, err = NewDynamoDbFixtureWriter(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("failed to create dynamodb fixture writer for %s: %w", settings.ModelId.String(), err)
		}

		return NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewDynamoDbFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, settings *ddb.Settings, options ...DdbWriterOption) (FixtureWriter, error) {
	ddbSettings := &ddb.Settings{
		ModelId: settings.ModelId,
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

	var err error
	var repo ddb.Repository
	var purger *dynamodbPurger

	if repo, err = ddb.NewRepository(ctx, config, logger, ddbSettings); err != nil {
		return nil, fmt.Errorf("failed to create dynamodb repository: %w", err)
	}

	if purger, err = NewDynamodbPurger(ctx, config, logger, ddbSettings); err != nil {
		return nil, fmt.Errorf("can not create dynamodb purger: %w", err)
	}

	return NewDynamoDbFixtureWriterWithInterfaces(logger, repo, purger), nil
}

func NewDynamoDbFixtureWriterWithInterfaces(logger log.Logger, repo ddb.Repository, purger *dynamodbPurger) FixtureWriter {
	return &dynamoDbFixtureWriter{
		logger: logger,
		repo:   repo,
		purger: purger,
	}
}

func (d *dynamoDbFixtureWriter) Purge(ctx context.Context) error {
	return d.purger.Purge(ctx)
}

func (d *dynamoDbFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	if len(fixtures) == 0 {
		return nil
	}

	if _, err := d.repo.BatchPutItems(ctx, fixtures); err != nil {
		return err
	}

	d.logger.Info("loaded %d dynamodb fixtures", len(fixtures))

	return nil
}
