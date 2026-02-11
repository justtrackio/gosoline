//go:build fixtures

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/redis"
)

type DynamoDbExampleModel struct {
	Name  string `ddb:"key=hash"`
	Value string `ddb:"global=hash"`
}

type OrmFixtureExample struct {
	db_repo.Model
	Name *string
}

var autoNumbered = fixtures.NewNumberSequenceFrom(2)

func mysqlOrmFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	mysqlMetadata := &db_repo.Metadata{
		ModelId: mdl.ModelId{
			Name: "orm_fixture_example",
		},
		TableName: "orm_fixture_examples",
	}
	mysqlOrmWriter, err := db_repo.NewMysqlOrmFixtureWriter(ctx, config, logger, mysqlMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mysql orm writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*OrmFixtureExample]{
		&fixtures.NamedFixture[*OrmFixtureExample]{
			Name: "foo",
			Value: &OrmFixtureExample{
				Model: db_repo.Model{
					Id: autoNumbered.GetNext(),
				},
				Name: mdl.Box("example"),
			},
		},
	}, mysqlOrmWriter), nil
}

func mysqlPlainFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	mysqlPlainWriter, err := db.NewMysqlPlainFixtureWriter(ctx, config, logger, &db.MysqlPlainMetaData{
		TableName: "plain_fixture_example",
		Columns:   []string{"id", "name"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize mysql plain writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[db.MysqlPlainFixtureValues]{
		{
			Name:  "foo2",
			Value: db.MysqlPlainFixtureValues{1, "testName1"},
		},
		{
			Name:  "foo3",
			Value: db.MysqlPlainFixtureValues{2, "testName2"},
		},
	}, mysqlPlainWriter), nil
}

func dynamodbKvstoreFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	dynamoDbKvStoreWriter, err := kvstore.NewDynamoDbKvStoreFixtureWriter[DynamoDbExampleModel](ctx, config, logger, &mdl.ModelId{
		Name: "exampleModel",
		Env:  "dev",
		App:  "fixture-loader",
		Tags: map[string]string{
			"project": "gosoline",
			"family":  "example",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize dynamodb kvstore writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*kvstore.KvStoreFixture]{
		{
			Name: "kv_somekey",
			Value: &kvstore.KvStoreFixture{
				Key:   "SomeKey",
				Value: DynamoDbExampleModel{Name: "Some Name", Value: "Some Value"},
			},
		},
	}, dynamoDbKvStoreWriter), nil
}

func redisFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	redisWriter, err := redis.NewRedisFixtureWriter(ctx, config, logger, "default", redis.RedisOpSet)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize redis writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*redis.RedisFixture]{
		{
			Name: "redis_example",
			Value: &redis.RedisFixture{
				Key:    "example-key",
				Value:  "bar",
				Expiry: 1 * time.Hour,
			},
		},
	}, redisWriter), nil
}

func dynamodbFixtureSet(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
	dynamodbWriter, err := ddb.NewDynamoDbFixtureWriter(ctx, config, logger, &ddb.Settings{
		ModelId: mdl.ModelId{
			Name: "exampleModel",
			Env:  "dev",
			App:  "fixture-loader",
			Tags: map[string]string{
				"project": "gosoline",
				"family":  "example",
			},
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
	fileReaderFactory := blob.NewFileReader("../../../test/fixtures/blob/test_data/fixtures_test_data")

	blobWriter, err := blob.NewBlobFixtureWriter(ctx, config, logger, fileReaderFactory, "test")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize blob writer: %w", err)
	}

	return fixtures.NewSimpleFixtureSet(fixtures.NamedFixtures[*blob.BlobFixture]{}, blobWriter), nil
}

func fixtureSetsFactory(ctx context.Context, config cfg.Config, logger log.Logger, group string) ([]fixtures.FixtureSet, error) {
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

type mod struct {
	kernel.ForegroundModule
}

func (m *mod) Run(ctx context.Context) error {
	<-ctx.Done()

	return nil
}

func main() {
	app := application.Default(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithFixtureSetFactory("default", fixtureSetsFactory),
		application.WithModuleFactory("main", func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
			return &mod{}, nil
		}),
	)

	app.Run()
}
