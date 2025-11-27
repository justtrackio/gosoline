package db_repo_test

import (
	"testing"
	"time"

	goSqlMock "github.com/DATA-DOG/go-sqlmock"
	"github.com/justtrackio/gosoline/pkg/clock"
	db_repo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/tracing"
	"github.com/stretchr/testify/suite"
)

func TestMysqlOrmFixtureWriterTestSuite(t *testing.T) {
	suite.Run(t, new(MysqlOrmFixtureWriterTestSuite))
}

type TestFixtureModel struct {
	db_repo.Model
	Name string `gorm:"column:name"`
}

func (t *TestFixtureModel) GetId() *uint {
	return t.Id
}

var TestFixtureMetadata = db_repo.Metadata{
	ModelId: mdl.ModelId{
		Application: "application",
		Name:        "testFixture",
	},
	TableName:  "test_fixtures",
	PrimaryKey: "test_fixtures.id",
	Mappings: db_repo.FieldMappings{
		"testFixture.id":   db_repo.NewFieldMapping("test_fixtures.id"),
		"testFixture.name": db_repo.NewFieldMapping("test_fixtures.name"),
	},
}

type MysqlOrmFixtureWriterTestSuite struct {
	suite.Suite

	mock   goSqlMock.Sqlmock
	writer fixtures.FixtureWriter
}

func (s *MysqlOrmFixtureWriterTestSuite) SetupTest() {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	tracer := tracing.NewLocalTracer()

	db, clientMock, err := goSqlMock.New()
	s.NoError(err)

	orm, err := db_repo.NewOrmWithInterfaces(db, db_repo.OrmSettings{
		Driver: "mysql",
	})
	s.NoError(err)

	testClock := clock.NewFakeClockAt(time.Unix(1549964818, 0))

	repo := db_repo.NewWithInterfaces(logger, tracer, orm, testClock, TestFixtureMetadata)

	s.mock = clientMock
	s.writer = db_repo.NewMysqlFixtureWriterWithInterfaces(logger, &TestFixtureMetadata, repo, nil)
}

func (s *MysqlOrmFixtureWriterTestSuite) TestWriteEmpty() {
	err := s.writer.Write(s.T().Context(), []any{})
	s.NoError(err)

	// No expectations set, so ExpectationsWereMet should pass if nothing was called
	if err := s.mock.ExpectationsWereMet(); err != nil {
		s.Failf("there were unfulfilled expectations", err.Error())
	}
}

func (s *MysqlOrmFixtureWriterTestSuite) TestWriteSingleFixture() {
	id := mdl.Box(uint(1))

	fixtureSetFixtures := []any{
		&TestFixtureModel{
			Model: db_repo.Model{Id: id},
			Name:  "Bob",
		},
	}

	// Bulk insert uses metadata table name (test_fixtures)
	result := goSqlMock.NewResult(1, 1)
	s.mock.ExpectExec("INSERT INTO `test_fixtures`").WillReturnResult(result)

	err := s.writer.Write(s.T().Context(), fixtureSetFixtures)
	s.NoError(err)

	if err := s.mock.ExpectationsWereMet(); err != nil {
		s.Failf("there were unfulfilled expectations", err.Error())
	}
}

func (s *MysqlOrmFixtureWriterTestSuite) TestWriteMultipleFixtures() {
	id1 := mdl.Box(uint(1))
	id2 := mdl.Box(uint(2))

	fixtureSetFixtures := []any{
		&TestFixtureModel{
			Model: db_repo.Model{Id: id1},
			Name:  "Bob",
		},
		&TestFixtureModel{
			Model: db_repo.Model{Id: id2},
			Name:  "Alice",
		},
	}

	// Bulk insert uses metadata table name (test_fixtures)
	result := goSqlMock.NewResult(2, 2)
	s.mock.ExpectExec("INSERT INTO `test_fixtures`").WillReturnResult(result)

	err := s.writer.Write(s.T().Context(), fixtureSetFixtures)
	s.NoError(err)

	if err := s.mock.ExpectationsWereMet(); err != nil {
		s.Failf("there were unfulfilled expectations", err.Error())
	}
}

func (s *MysqlOrmFixtureWriterTestSuite) TestWriteWithChunking() {
	// Create a writer with small batch size to test chunking
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))
	tracer := tracing.NewLocalTracer()

	db, clientMock, err := goSqlMock.New()
	s.NoError(err)

	orm, err := db_repo.NewOrmWithInterfaces(db, db_repo.OrmSettings{
		Driver: "mysql",
	})
	s.NoError(err)

	testClock := clock.NewFakeClockAt(time.Unix(1549964818, 0))

	repo := db_repo.NewWithInterfaces(logger, tracer, orm, testClock, TestFixtureMetadata)

	// Use batch size of 2 to force chunking with 3 fixtures
	writer := db_repo.NewMysqlFixtureWriterWithInterfaces(logger, &TestFixtureMetadata, repo, &db_repo.MysqlOrmSettings{
		BatchSize: 2,
	})

	id1 := mdl.Box(uint(1))
	id2 := mdl.Box(uint(2))
	id3 := mdl.Box(uint(3))

	fixtureSetFixtures := []any{
		&TestFixtureModel{
			Model: db_repo.Model{Id: id1},
			Name:  "Bob",
		},
		&TestFixtureModel{
			Model: db_repo.Model{Id: id2},
			Name:  "Alice",
		},
		&TestFixtureModel{
			Model: db_repo.Model{Id: id3},
			Name:  "Charlie",
		},
	}

	// Bulk insert with chunking uses metadata table name (test_fixtures)
	// First chunk: 2 fixtures
	result1 := goSqlMock.NewResult(2, 2)
	clientMock.ExpectExec("INSERT INTO `test_fixtures`").WillReturnResult(result1)

	// Second chunk: 1 fixture
	result2 := goSqlMock.NewResult(1, 1)
	clientMock.ExpectExec("INSERT INTO `test_fixtures`").WillReturnResult(result2)

	err = writer.Write(s.T().Context(), fixtureSetFixtures)
	s.NoError(err)

	if err := clientMock.ExpectationsWereMet(); err != nil {
		s.Failf("there were unfulfilled expectations", err.Error())
	}
}
