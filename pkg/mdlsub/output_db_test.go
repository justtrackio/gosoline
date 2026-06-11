package mdlsub_test

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	dbPkg "github.com/justtrackio/gosoline/pkg/db"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/stretchr/testify/suite"
)

type outputDbTestModel struct {
	Id   uint `gorm:"primary_key"`
	Name string
}

func (m outputDbTestModel) GetId() any {
	return m.Id
}

func (m outputDbTestModel) TableName() string {
	return "output_db_test_models"
}

func TestOutputDbTestSuite(t *testing.T) {
	suite.Run(t, new(OutputDbTestSuite))
}

type OutputDbTestSuite struct {
	suite.Suite
	db     *sql.DB
	mock   sqlmock.Sqlmock
	output *mdlsub.OutputDb
	model  outputDbTestModel
}

func (s *OutputDbTestSuite) SetupTest() {
	db, mock, err := sqlmock.New()
	s.Require().NoError(err)

	orm, err := dbRepo.NewOrmWithInterfaces(db, dbRepo.OrmSettings{
		Driver: "mysql",
	})
	s.Require().NoError(err)

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(s.T()))

	s.db = db
	s.mock = mock
	s.output = mdlsub.NewOutputDbWithInterfaces(logger, orm)
	s.model = outputDbTestModel{
		Id:   1,
		Name: "test",
	}
}

func (s *OutputDbTestSuite) TearDownTest() {
	s.mock.ExpectClose()
	s.NoError(s.db.Close())
	s.NoError(s.mock.ExpectationsWereMet())
}

// expectUpdate queues the BEGIN/UPDATE/COMMIT gorm runs for a Save, returning the
// given number of affected rows.
func (s *OutputDbTestSuite) expectUpdate(rowsAffected int64) {
	s.mock.ExpectBegin()
	s.mock.ExpectExec("UPDATE `output_db_test_models` SET `name` = \\? WHERE `output_db_test_models`\\.`id` = \\?").
		WithArgs("test", 1).
		WillReturnResult(sqlmock.NewResult(0, rowsAffected))
	s.mock.ExpectCommit()
}

// expectSelect queues the SELECT gorm runs after an UPDATE affected no rows. When
// found is false the row is reported as missing, so gorm falls back to an INSERT.
func (s *OutputDbTestSuite) expectSelect(found bool) {
	rows := sqlmock.NewRows([]string{"id", "name"})
	if found {
		rows.AddRow(1, "test")
	}

	s.mock.ExpectQuery("SELECT \\* FROM `output_db_test_models` WHERE `output_db_test_models`\\.`id` = \\? ORDER BY `output_db_test_models`\\.`id` ASC LIMIT 1").
		WithArgs(1).
		WillReturnRows(rows)
}

// expectInsertDuplicate queues the BEGIN/INSERT/ROLLBACK gorm runs for the
// FirstOrCreate fallback, with the INSERT failing on a duplicate-entry error.
func (s *OutputDbTestSuite) expectInsertDuplicate(key string) {
	s.mock.ExpectBegin()
	s.mock.ExpectExec("INSERT INTO `output_db_test_models`").
		WithArgs(1, "test").
		WillReturnError(&mysql.MySQLError{Number: mysqlerr.ER_DUP_ENTRY, Message: "Duplicate entry '1' for key '" + key + "'"})
	s.mock.ExpectRollback()
}

func (s *OutputDbTestSuite) TestPersistUpdateWithNoRowsAffectedDoesNotPanic() {
	s.expectUpdate(0)
	s.expectSelect(true)

	s.NotPanics(func() {
		s.NoError(s.output.Persist(s.T().Context(), s.model, dbRepo.Update))
	})
}

func (s *OutputDbTestSuite) TestPersistWithPointerModelReturnsError() {
	err := s.output.Persist(s.T().Context(), &s.model, dbRepo.Update)
	s.EqualError(err, "model must not be a pointer")
}

func (s *OutputDbTestSuite) TestPersistWithNilModelReturnsError() {
	err := s.output.Persist(s.T().Context(), nil, dbRepo.Update)
	s.EqualError(err, "model must not be nil")
}

func (s *OutputDbTestSuite) TestPersistRetriesOnDuplicateEntry() {
	// First attempt: a concurrent writer inserts the same key between SELECT and
	// INSERT, so the INSERT fails with 1062.
	s.expectUpdate(0)
	s.expectSelect(false)
	s.expectInsertDuplicate("PRIMARY")

	// Retry: the row now exists, so the UPDATE matches and Save succeeds.
	s.expectUpdate(1)

	s.NoError(s.output.Persist(s.T().Context(), s.model, dbRepo.Update))
}

func (s *OutputDbTestSuite) TestPersistReturnsErrorWhenDuplicateEntryPersists() {
	// Every attempt fails with an unrecoverable duplicate (e.g. a secondary unique
	// key). Persist makes 1+MaxPersistRetries attempts, then surfaces the error.
	for i := 0; i <= mdlsub.MaxPersistRetries; i++ {
		s.expectUpdate(0)
		s.expectSelect(false)
		s.expectInsertDuplicate("name")
	}

	err := s.output.Persist(s.T().Context(), s.model, dbRepo.Update)
	s.Error(err)
	s.True(dbPkg.IsDuplicateEntryError(err), "expected a duplicate entry error, got: %v", err)
}
