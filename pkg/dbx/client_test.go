package dbx_test

import (
	"context"
	"testing"

	goSqlMock "github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/dbx"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TestEntity struct {
	Id   int    `db:"id"`
	Name string `db:"name"`
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
		assert.NoError(s.T(), err)
	}
	loggerMock := logMocks.NewLoggerMock(logMocks.WithMockAll)
	sqlxDB := sqlx.NewDb(dbMock, "sqlmock")

	dbClient := db.NewClientWithInterfaces(loggerMock, sqlxDB, exec.NewDefaultExecutor())

	s.sqlMock = sqlMock
	s.client = dbx.NewClientWithInterfaces[TestEntity](dbClient, "test_table")
}

func (s *ClientTestSuite) TestDelete() {
	ctx := s.T().Context()

	s.sqlMock.
		ExpectExec("DELETE FROM test_table WHERE id = ?").
		WithArgs(1).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	result, err := s.client.Delete().Where(dbx.Eq{"id": 1}).Exec(ctx)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
}

func (s *ClientTestSuite) TestInsert() {
	ctx := s.T().Context()

	testEntity := TestEntity{
		Id:   1,
		Name: "Test Name",
	}

	s.sqlMock.
		ExpectExec("INSERT INTO test_table (id,name) VALUES (?,?)").
		WithArgs(testEntity.Id, testEntity.Name).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	result, err := s.client.Insert(testEntity).Exec(ctx)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
}

func (s *ClientTestSuite) TestReplace() {
	ctx := s.T().Context()

	testEntity := TestEntity{
		Id:   1,
		Name: "Test Name",
	}

	s.sqlMock.
		ExpectExec("REPLACE INTO test_table (id,name) VALUES (?,?)").
		WithArgs(testEntity.Id, testEntity.Name).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	result, err := s.client.Replace(testEntity).Exec(ctx)

	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), result)
}

func (s *ClientTestSuite) TestSelect() {
	testEntity := TestEntity{
		Id:   1,
		Name: "Test Name",
	}

	s.sqlMock.
		ExpectQuery("SELECT id, name FROM test_table WHERE id = ?").
		WillReturnRows(goSqlMock.NewRows([]string{"id", "name"}).AddRow(testEntity.Id, testEntity.Name))

	_, err := s.client.Select().Where(dbx.Eq{"id": 1}).Exec(context.Background())
	assert.NoError(s.T(), err)
}

func (s *ClientTestSuite) TestSelectWhereStruct() {
	testEntity := TestEntity{
		Id:   1,
		Name: "Test Name",
	}

	s.sqlMock.
		ExpectQuery("SELECT id, name FROM test_table WHERE id = ?").
		WithArgs(1).
		WillReturnRows(goSqlMock.NewRows([]string{"id", "name"}).
			AddRow(testEntity.Id, testEntity.Name))

	todos, err := s.client.Select().Where(TestEntity{Id: 1}).Exec(context.Background())
	s.NoError(err)
	s.Len(todos, 1)
}

func (s *ClientTestSuite) TestUpdate() {
	s.sqlMock.
		ExpectExec("UPDATE test_table SET name = ? WHERE id = ?").
		WithArgs("foobar", 1).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	_, err := s.client.Update().Set("name", "foobar").Where(dbx.Eq{"id": 1}).Exec(context.Background())
	assert.NoError(s.T(), err)
}

func (s *ClientTestSuite) TestUpdateStruct() {
	s.sqlMock.
		ExpectExec("UPDATE test_table SET name = ? WHERE id = ?").
		WithArgs("foobar", 1).
		WillReturnResult(goSqlMock.NewResult(1, 1))

	_, err := s.client.Update(TestEntity{Name: "foobar"}).Where(TestEntity{Id: 1}).Exec(context.Background())
	assert.NoError(s.T(), err)
}
