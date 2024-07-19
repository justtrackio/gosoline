//go:build fixtures

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type DynamoDbExampleModel struct {
	Name  string `ddb:"key=hash"`
	Value string `ddb:"global=hash"`
}

type OrmFixtureExample struct {
	db_repo.Model
	Name *string
}

var autoNumbered = fixtures.NewAutoNumberedFrom(2)

func mysqlOrmFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	mysqlMetadata := &db_repo.Metadata{
		ModelId: mdl.ModelId{
			Name: "orm_fixture_example",
		},
	}
	mysqlOrmWriter, err := fixtures.NewMysqlOrmFixtureWriter(ctx, config, logger, mysqlMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mysql orm writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[OrmFixtureExample]{
		&fixtures.NamedFixture[OrmFixtureExample]{
			Name: "foo",
			Value: OrmFixtureExample{
				Model: db_repo.Model{
					Id: autoNumbered.GetNext(),
				},
				Name: mdl.Box("example"),
			},
		},
	}, mysqlOrmWriter), nil
}

func mysqlPlainFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	mysqlPlainWriter, err := fixtures.NewMysqlPlainFixtureWriter(ctx, config, logger, &fixtures.MysqlPlainMetaData{
		TableName: "plain_fixture_example",
		Columns:   []string{"id", "name"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mysql plain writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[fixtures.MysqlPlainFixtureValues]{
		{
			Name:  "foo2",
			Value: fixtures.MysqlPlainFixtureValues{1, "testName1"},
		},
		{
			Name:  "foo3",
			Value: fixtures.MysqlPlainFixtureValues{2, "testName2"},
		},
	}, mysqlPlainWriter), nil
}

func dynamodbKvstoreFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	dynamoDbKvStoreWriter, err := fixtures.NewDynamoDbKvStoreFixtureWriter[DynamoDbExampleModel](ctx, config, logger, &mdl.ModelId{
		Project:     "gosoline",
		Environment: "dev",
		Family:      "example",
		Application: "fixture-loader",
		Name:        "exampleModel",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dynamodb kvstore writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*fixtures.KvStoreFixture]{
		{
			Name: "kv_somekey",
			Value: &fixtures.KvStoreFixture{
				Key:   "SomeKey",
				Value: DynamoDbExampleModel{Name: "Some Name", Value: "Some Value"},
			},
		},
	}, dynamoDbKvStoreWriter), nil
}

func redisFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	redisWriter, err := fixtures.NewRedisFixtureWriter(ctx, config, logger, "default", fixtures.RedisOpSet)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*fixtures.RedisFixture]{
		{
			Name: "redis_example",
			Value: &fixtures.RedisFixture{
				Key:    "example-key",
				Value:  "bar",
				Expiry: 1 * time.Hour,
			},
		},
	}, redisWriter), nil
}

func dynamodbFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	dynamodbWriter, err := fixtures.NewDynamoDbFixtureWriter(ctx, config, logger, &ddb.Settings{
		ModelId: mdl.ModelId{
			Project:     "gosoline",
			Environment: "dev",
			Family:      "example",
			Application: "fixture-loader",
			Name:        "exampleModel",
		},
		Main: ddb.MainSettings{
			Model: DynamoDbExampleModel{},
		},
		Global: []ddb.GlobalSettings{
			{
				Name:               "IDX_Name",
				Model:              DynamoDbExampleModel{},
				ReadCapacityUnits:  1,
				WriteCapacityUnits: 1,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dynamodb writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*DynamoDbExampleModel]{
		{
			Name:  "ddb",
			Value: &DynamoDbExampleModel{Name: "Some Name", Value: "Some Value"},
		},
	}, dynamodbWriter), nil
}

func blobFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	blobWriter, err := fixtures.NewBlobFixtureWriter(ctx, config, logger, &fixtures.BlobFixturesSettings{
		ConfigName: "test",
		BasePath:   "../../test/test_data/s3_fixtures_test_data",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize blob writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*fixtures.BlobFixture]{}, blobWriter), nil
}

func fixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger) ([]fixtures.FixtureSet, error) {
	mysqlOrmFs, err := mysqlOrmFixtureSet(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	mysqlPlainFs, err := mysqlPlainFixtureSet(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	dynamodbKvstoreFs, err := dynamodbKvstoreFixtureSet(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	redisFs, err := redisFixtureSet(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	dynamodbFs, err := dynamodbFixtureSet(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	blobFs, err := blobFixtureSet(ctx, config, logger)
	if err != nil {
		return nil, err
	}

	return []fixtures.FixtureSet{
		mysqlOrmFs,
		mysqlPlainFs,
		dynamodbKvstoreFs,
		redisFs,
		dynamodbFs,
		blobFs,
	}, nil
}

func main() {
	app := application.Default(
		application.WithFixtureSetFactory(fixtureSetsFactory),
	)

	app.Run()
}
