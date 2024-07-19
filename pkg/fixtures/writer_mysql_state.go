package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
)

type newMysqlStateServiceKey int

const mysqlStateCreate = "CREATE TABLE IF NOT EXISTS fixture (id int unsigned auto_increment primary key, local_table_name varchar(64), data_set_db_name varchar(64), created_at datetime not null, updated_at datetime not null)"

var mysqlStateMetadata = db_repo.Metadata{
	TableName: "fixture",
}

type mysqlStateService struct {
	repo db_repo.Repository
}

func provideMysqlStateService(ctx context.Context, config cfg.Config, logger log.Logger) (*mysqlStateService, error) {
	return appctx.Provide(ctx, newMysqlStateServiceKey(1), func() (*mysqlStateService, error) {
		return newMysqlStateService(ctx, config, logger)
	})
}

func newMysqlStateService(ctx context.Context, config cfg.Config, logger log.Logger) (*mysqlStateService, error) {
	repo, err := db_repo.New(ctx, config, logger, db_repo.Settings{
		Metadata: mysqlStateMetadata,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create fixture repo: %w", err)
	}

	// ensure state table exists
	dbClient, err := db.ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("failed to provide db client: %w", err)
	}

	rows, err := dbClient.Query(ctx, mysqlStateCreate)
	if err != nil {
		return nil, fmt.Errorf("failed to create mysql fixture state table: %w", err)
	}
	// /ensure state table exists

	if err = rows.Close(); err != nil {
		return nil, fmt.Errorf("failed to close mysql state statement: %w", err)
	}

	return &mysqlStateService{
		repo: repo,
	}, nil
}

func (m *mysqlStateService) Get(ctx context.Context, table string) (*mysqlStateFixture, error) {
	mysqlStateFixtures := make([]*mysqlStateFixture, 0)
	qb := db_repo.NewQueryBuilder()
	qb.Where(&mysqlStateFixture{
		LocalTableName: table,
	})

	if err := m.repo.Query(ctx, qb, &mysqlStateFixtures); err != nil {
		return nil, fmt.Errorf("could not fetch mysql fixtures state: %w", err)
	}

	if len(mysqlStateFixtures) == 0 {
		return nil, nil
	}

	return mysqlStateFixtures[0], nil
}

func (m *mysqlStateService) Persist(ctx context.Context, dataSetDbName string, table string) (*mysqlStateFixture, error) {
	mFixture, err := m.Get(ctx, table)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current data set db name: %w", err)
	}

	if mFixture == nil {
		mFixture = newMysqlStateFixture(table, dataSetDbName)
	}

	mFixture.DataSetDbName = dataSetDbName

	err = m.repo.Update(ctx, mFixture)
	if err != nil {
		return nil, fmt.Errorf("failed to save current data set db name: %w", err)
	}

	return mFixture, nil
}
