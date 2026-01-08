package dbx_test

import (
	"database/sql/driver"
	"testing"

	goSqlMock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/dbx"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/suite"
)

type TestEntity struct {
	Id      int    `db:"id"`
	Name    string `db:"name"`
	Enabled bool   `db:"enabled"`
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

type ClientTestSuite struct {
	suite.Suite

	sqlMock goSqlMock.Sqlmock
	client  dbx.Client[TestEntity]
}

func (s *ClientTestSuite) SetupTest() {
	dbMock, sqlMock, err := goSqlMock.New(goSqlMock.QueryMatcherOption(goSqlMock.QueryMatcherEqual))
	if err != nil {
		s.Error(err)
	}
	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	sqlxDB := sqlx.NewDb(dbMock, "sqlmock")

	dbClient := db.NewClientWithInterfaces(loggerMock, sqlxDB, exec.NewDefaultExecutor())

	s.sqlMock = sqlMock
	s.client, err = dbx.NewClientWithInterfaces[TestEntity](dbClient, "test_table", dbx.Question)
	s.NoError(err, "should create a new dbx client without error")
}

func (s *ClientTestSuite) TestDelete() {
	ctx := s.T().Context()

	s.sqlMock.
		ExpectExec("DELETE FROM test_table WHERE id = ?").
		WithArgs(1).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	result, err := s.client.Delete().Where(dbx.Eq{"id": 1}).Exec(ctx)

	s.NoError(err)
	s.NotNil(result)
}

func (s *ClientTestSuite) TestDeleteTyped() {
	ctx := s.T().Context()

	s.sqlMock.
		ExpectExec("DELETE FROM test_table WHERE id = ?").
		WithArgs(1).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	result, err := s.client.Delete().Where(TestEntity{Id: 1}).Exec(ctx)

	s.NoError(err)
	s.NotNil(result)
}

func (s *ClientTestSuite) TestDeleteOptions() {
	ctx := s.T().Context()

	s.sqlMock.
		ExpectExec("DELETE LOW_PRIORITY FROM test_table").
		WillReturnResult(goSqlMock.NewResult(1, 1))

	result, err := s.client.Delete().Options("LOW_PRIORITY").Exec(ctx)

	s.NoError(err)
	s.NotNil(result)
}

func (s *ClientTestSuite) TestInsert() {
	ctx := s.T().Context()

	testEntity := TestEntity{
		Id:   1,
		Name: "Test Name",
	}

	s.sqlMock.
		ExpectExec("INSERT INTO test_table (`id`,`name`,`enabled`) VALUES (?,?,?)").
		WithArgs(testEntity.Id, testEntity.Name, testEntity.Enabled).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	result, err := s.client.Insert(testEntity).Exec(ctx)

	s.NoError(err)
	s.NotNil(result)
}

func (s *ClientTestSuite) TestInsertBatch() {
	ctx := s.T().Context()

	testEntities := []TestEntity{
		{Id: 1, Name: "Test Name"},
		{Id: 2, Name: "foo"},
		{Id: 3, Name: "bar"},
	}
	args := []driver.Value{
		testEntities[0].Id, testEntities[0].Name, testEntities[0].Enabled,
		testEntities[1].Id, testEntities[1].Name, testEntities[1].Enabled,
		testEntities[2].Id, testEntities[2].Name, testEntities[2].Enabled,
	}

	s.sqlMock.
		ExpectExec("INSERT INTO test_table (`id`,`name`,`enabled`) VALUES (?,?,?),(?,?,?),(?,?,?)").
		WithArgs(args...).
		WillReturnResult(goSqlMock.NewResult(3, 3))

	result, err := s.client.Insert(testEntities...).Exec(ctx)

	s.NoError(err)
	s.NotNil(result)
}

func (s *ClientTestSuite) TestReplace() {
	ctx := s.T().Context()

	testEntity := TestEntity{
		Id:      1,
		Name:    "Test Name",
		Enabled: true,
	}

	s.sqlMock.
		ExpectExec("REPLACE INTO test_table (`id`,`name`,`enabled`) VALUES (?,?,?)").
		WithArgs(testEntity.Id, testEntity.Name, testEntity.Enabled).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	result, err := s.client.Replace(testEntity).Exec(ctx)

	s.NoError(err)
	s.NotNil(result)
}

func (s *ClientTestSuite) TestSelect() {
	testEntity := TestEntity{
		Id:   1,
		Name: "Test Name",
	}

	s.sqlMock.
		ExpectQuery("SELECT `id`, `name`, `enabled` FROM test_table WHERE id = ?").
		WillReturnRows(goSqlMock.NewRows([]string{"id", "name"}).AddRow(testEntity.Id, testEntity.Name))

	_, err := s.client.Select().Where(dbx.Eq{"id": 1}).Exec(s.T().Context())
	s.NoError(err)
}

func (s *ClientTestSuite) TestSelectWhereStruct() {
	testEntity := TestEntity{
		Id:   1,
		Name: "Test Name",
	}

	s.sqlMock.
		ExpectQuery("SELECT `id`, `name`, `enabled` FROM test_table WHERE id = ?").
		WithArgs(1).
		WillReturnRows(goSqlMock.NewRows([]string{"id", "name"}).
			AddRow(testEntity.Id, testEntity.Name))

	todos, err := s.client.Select().Where(TestEntity{Id: 1}).Exec(s.T().Context())
	s.NoError(err)
	s.Len(todos, 1)
}

func (s *ClientTestSuite) TestSelectOptions() {
	testEntity := TestEntity{
		Id:   1,
		Name: "Test Name",
	}

	s.sqlMock.
		ExpectQuery("SELECT SQL_NO_CACHE HIGH_PRIORITY `id`, `name`, `enabled` FROM test_table").
		WillReturnRows(goSqlMock.NewRows([]string{"id", "name"}).AddRow(testEntity.Id, testEntity.Name))

	_, err := s.client.Select().Options("SQL_NO_CACHE", "HIGH_PRIORITY").Exec(s.T().Context())
	s.NoError(err)
}

func (s *ClientTestSuite) TestSelectJoinGroupBy() {
	testEntity := TestEntity{
		Id:   1,
		Name: "Test Name",
	}

	s.sqlMock.
		ExpectQuery("SELECT MAX(id) AS id, MIN(name) AS name FROM test_table JOIN join_table AS jt ON jt.id = test_table.id GROUP BY group_column").
		WillReturnRows(goSqlMock.NewRows([]string{"id", "name"}).AddRow(testEntity.Id, testEntity.Name))

	res, err := s.client.Select().
		Column("MAX(id) AS id").
		Column("MIN(name) AS name").
		Join("join_table AS jt ON jt.id = test_table.id").
		GroupBy("group_column").
		Exec(s.T().Context())

	s.NoError(err)
	s.Equal(testEntity, res[0])
}

func (s *ClientTestSuite) TestUpdate() {
	s.sqlMock.
		ExpectExec("UPDATE test_table SET name = ? WHERE id = ? ORDER BY id ASC LIMIT 2").
		WithArgs("foobar", 1).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	_, err := s.client.Update().
		Set("name", "foobar").
		Where(dbx.Eq{"id": 1}).
		OrderBy("id ASC").
		Limit(2).
		Exec(s.T().Context())
	s.NoError(err)
}

func (s *ClientTestSuite) TestUpdateMap() {
	s.sqlMock.
		ExpectExec("UPDATE test_table SET name = ? WHERE id = ?").
		WithArgs("foobar", 1).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	_, err := s.client.Update(map[string]any{
		"name": "foobar",
	}).Where(dbx.Eq{"id": 1}).Exec(s.T().Context())
	s.NoError(err)
}

func (s *ClientTestSuite) TestUpdateStruct() {
	s.sqlMock.
		ExpectExec("UPDATE test_table SET name = ? WHERE id = ?").
		WithArgs("foobar", 1).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	_, err := s.client.Update(TestEntity{Name: "foobar"}).Where(TestEntity{Id: 1}).Exec(s.T().Context())
	s.NoError(err)
}

func (s *ClientTestSuite) TestUpdateStructAndMap() {
	s.sqlMock.
		ExpectExec("UPDATE test_table SET enabled = ?, name = ? WHERE id = ?").
		WithArgs(false, "foobar", 1).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	updateInputs := []any{
		TestEntity{Name: "foobar"},
		map[string]any{"enabled": false},
	}

	_, err := s.client.Update(updateInputs...).Where(TestEntity{Id: 1}).Exec(s.T().Context())
	s.NoError(err)
}

func (s *ClientTestSuite) TestUpdateInvalidInput() {
	_, err := s.client.Update("invalid input").Where(TestEntity{Id: 1}).Exec(s.T().Context())
	s.Error(err, "unable to execute update query: unsupported type string for update values")
}

func (s *ClientTestSuite) TestUpdateOptions() {
	s.sqlMock.
		ExpectExec("UPDATE IGNORE test_table SET name = ?").
		WithArgs("foobar").
		WillReturnResult(goSqlMock.NewResult(1, 1))

	_, err := s.client.Update().Options("IGNORE").Set("name", "foobar").Exec(s.T().Context())
	s.NoError(err)
}

func (s *ClientTestSuite) TestGet() {
	ctx := s.T().Context()

	testEntity := TestEntity{
		Id:      1,
		Name:    "Test Name",
		Enabled: true,
	}

	s.sqlMock.
		ExpectQuery("SELECT `id`, `name`, `enabled` FROM test_table WHERE id = ? LIMIT 2").
		WithArgs(1).
		WillReturnRows(
			goSqlMock.NewRows([]string{"id", "name", "enabled"}).
				AddRow(testEntity.Id, testEntity.Name, testEntity.Enabled),
		)

	res, err := s.client.Get().Where(dbx.Eq{"id": 1}).Exec(ctx)

	s.NoError(err)
	s.Equal(testEntity, res)
}

func (s *ClientTestSuite) TestGetWhereStruct() {
	ctx := s.T().Context()

	testEntity := TestEntity{
		Id:      1,
		Name:    "Test Name",
		Enabled: true,
	}

	s.sqlMock.
		ExpectQuery("SELECT `id`, `name`, `enabled` FROM test_table WHERE id = ? LIMIT 2").
		WithArgs(1).
		WillReturnRows(
			goSqlMock.NewRows([]string{"id", "name", "enabled"}).
				AddRow(testEntity.Id, testEntity.Name, testEntity.Enabled),
		)

	res, err := s.client.Get().Where(TestEntity{Id: 1}).Exec(ctx)

	s.NoError(err)
	s.Equal(testEntity, res)
}

func (s *ClientTestSuite) TestGetNotFound() {
	ctx := s.T().Context()

	s.sqlMock.
		ExpectQuery("SELECT `id`, `name`, `enabled` FROM test_table WHERE id = ? LIMIT 2").
		WithArgs(1).
		WillReturnRows(
			goSqlMock.NewRows([]string{"id", "name", "enabled"}),
		)

	_, err := s.client.Get().Where(dbx.Eq{"id": 1}).Exec(ctx)

	s.ErrorIs(err, dbx.ErrNotFound)
}

func (s *ClientTestSuite) TestGetTooManyResults() {
	ctx := s.T().Context()

	s.sqlMock.
		ExpectQuery("SELECT `id`, `name`, `enabled` FROM test_table WHERE id = ? LIMIT 2").
		WithArgs(1).
		WillReturnRows(
			goSqlMock.NewRows([]string{"id", "name", "enabled"}).
				AddRow(1, "a", true).
				AddRow(1, "b", false),
		)

	_, err := s.client.Get().Where(dbx.Eq{"id": 1}).Exec(ctx)

	s.Error(err)
	s.Contains(err.Error(), "expected 1 result, got 2")
}
