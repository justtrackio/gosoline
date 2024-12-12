package athena_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Masterminds/squirrel"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/athena"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/stretchr/testify/suite"
)

func TestAthenaRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(AthenaRepositoryTestSuite))
}

type TestValue struct {
	Id        int       `db:"id"`
	Name      string    `db:"name"`
	CreatedAt time.Time `db:"created_at"`
}

type AthenaRepositoryTestSuite struct {
	suite.Suite

	ctx        context.Context
	now        time.Time
	clock      clock.Clock
	sqlMock    sqlmock.Sqlmock
	repository athena.Repository[TestValue]
}

func (s *AthenaRepositoryTestSuite) SetupSuite() {
	settings := &athena.Settings{TableName: "testSchema"}

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	raw := athena.NewRepositoryRawWithInterfaces(db, exec.NewDefaultExecutor(), settings)
	s.NoError(err)

	s.ctx = context.Background()
	s.now = time.Unix(1707402132, 0)
	s.clock = clock.NewFakeClockAt(s.now)
	s.sqlMock = mock
	s.repository = athena.NewRepositoryWithInterfaces[TestValue](raw, &athena.Settings{TableName: "testSchema"})
}

func (s *AthenaRepositoryTestSuite) TestQueryBuilder() {
	expectedQry := squirrel.Select("id", "name", "created_at").From("testSchema")
	actualQry := s.repository.QueryBuilder()

	s.Equal(expectedQry, actualQry)
}

func (s *AthenaRepositoryTestSuite) TestRunQueryQb() {
	rows := sqlmock.NewRows([]string{"id", "name", "created_at"})
	rows.AddRow(1, "foo", s.now)

	s.sqlMock.ExpectQuery(`SELECT id, name, created_at FROM testSchema WHERE (id = 1 AND name = 'foo')`).WillReturnRows(rows)

	expectedResult := []TestValue{
		{1, "foo", s.now},
	}

	qb := s.repository.QueryBuilder().Where(squirrel.And{squirrel.Eq{"id": 1}, squirrel.Eq{"name": "foo"}})

	actualResult, err := s.repository.Query(s.ctx, qb)
	s.NoError(err)
	s.Equal(expectedResult, actualResult)
}

func (s *AthenaRepositoryTestSuite) TestRunQuery() {
	rows := sqlmock.NewRows([]string{"id", "name", "created_at"})
	rows.AddRow(1, "foo", s.now)
	rows.AddRow(2, "bar", s.now)

	s.sqlMock.ExpectQuery("SELECT id, name, created_at FROM testSchema").WillReturnRows(rows)

	expectedResult := []TestValue{
		{1, "foo", s.now},
		{2, "bar", s.now},
	}

	actualResult, err := s.repository.QuerySql(s.ctx, "SELECT id, name, created_at FROM testSchema")
	s.NoError(err)
	s.Equal(expectedResult, actualResult)
}
