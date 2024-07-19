package fixtures_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/suite"
)

func TestMysqlSqlxFixtureWriterTestSuite(t *testing.T) {
	suite.Run(t, new(MysqlSqlxFixtureWriterTestSuite))
}

type MysqlSqlxFixtureWriterTestSuite struct {
	suite.Suite

	mock   sqlmock.Sqlmock
	writer fixtures.FixtureWriter
}

func (s *MysqlSqlxFixtureWriterTestSuite) SetupSuite() {
	var err error
	var sdb *sql.DB
	var client db.Client

	sdb, s.mock, err = sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	s.NoError(err)

	xdb := sqlx.NewDb(sdb, "mysql")
	logger := mocks.NewLoggerMockedAll()

	client = db.NewClientWithInterfaces(logger, xdb, exec.NewDefaultExecutor())
	s.writer = fixtures.NewMysqlSqlxFixtureWriterWithInterfaces(logger, client, &fixtures.MysqlSqlxMetaData{TableName: "table"}, nil)
}

func (s *MysqlSqlxFixtureWriterTestSuite) TestWrite() {
	type Fixture struct {
		Id       int    `db:"id"`
		Name     string `db:"name"`
		IsActive bool   `db:"is_active"`
	}

	fixtureSetFixtures := []interface{}{
		Fixture{
			Id:       1,
			Name:     "Bob",
			IsActive: true,
		},
		Fixture{
			Id:       2,
			Name:     "Alice",
			IsActive: false,
		},
	}

	s.mock.ExpectExec(`INSERT INTO table (id,name,is_active) VALUES (?,?,?)`).
		WithArgs(1, "Bob", true).
		WillReturnResult(sqlmock.NewResult(0, 1))
	s.mock.ExpectExec(`INSERT INTO table (id,name,is_active) VALUES (?,?,?)`).
		WithArgs(2, "Alice", false).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := s.writer.Write(context.Background(), fixtureSetFixtures)
	s.NoError(err)

	if err := s.mock.ExpectationsWereMet(); err != nil {
		s.Failf("there were unfulfilled expectations: %s", err.Error())
	}
}
