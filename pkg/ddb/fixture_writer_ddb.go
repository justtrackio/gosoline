package ddb

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

type dynamoDbFixtureWriter struct {
	logger log.Logger
	repo   Repository
}

func DynamoDbFixtureSetFactory[T any](settings *Settings, data fixtures.NamedFixtures[T], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewDynamoDbFixtureWriter(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("failed to create dynamodb fixture writer for %s: %w", settings.ModelId.String(), err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewDynamoDbFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings, options ...DdbWriterOption) (fixtures.FixtureWriter, error) {
	ddbSettings := &Settings{
		ModelId: settings.ModelId,
		Main: MainSettings{
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
	var repo Repository

	if repo, err = NewRepository(ctx, config, logger, ddbSettings); err != nil {
		return nil, fmt.Errorf("failed to create dynamodb repository: %w", err)
	}

	return NewDynamoDbFixtureWriterWithInterfaces(logger, repo), nil
}

func NewDynamoDbFixtureWriterWithInterfaces(logger log.Logger, repo Repository) fixtures.FixtureWriter {
	return &dynamoDbFixtureWriter{
		logger: logger,
		repo:   repo,
	}
}

func (d *dynamoDbFixtureWriter) Write(ctx context.Context, fixtures []any) error {
	if len(fixtures) == 0 {
		return nil
	}

	if _, err := d.repo.BatchPutItems(ctx, fixtures); err != nil {
		return err
	}

	d.logger.Info(ctx, "loaded %d dynamodb fixtures", len(fixtures))

	return nil
}
