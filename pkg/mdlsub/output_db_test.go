package mdlsub

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestOutputDbPersistUpdateWithNoRowsAffectedDoesNotPanic(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	orm, err := db_repo.NewOrmWithInterfaces(db, db_repo.OrmSettings{
		Driver: "mysql",
	})
	require.NoError(t, err)

	output := &OutputDb{
		orm: orm,
	}
	model := outputDbTestModel{
		Id:   1,
		Name: "test",
	}

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `output_db_test_models` SET `name` = \\? WHERE `output_db_test_models`\\.`id` = \\?").
		WithArgs("test", 1).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	mock.ExpectQuery("SELECT \\* FROM `output_db_test_models` WHERE `output_db_test_models`\\.`id` = \\? ORDER BY `output_db_test_models`\\.`id` ASC LIMIT 1").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow(1, "test"))
	mock.ExpectClose()

	assert.NotPanics(t, func() {
		err = output.Persist(t.Context(), model, db_repo.Update)
	})
	assert.NoError(t, err)
	assert.NoError(t, db.Close())
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestOutputDbPersistWithPointerModelReturnsError(t *testing.T) {
	output := &OutputDb{}
	err := output.Persist(t.Context(), &outputDbTestModel{Id: 1, Name: "test"}, db_repo.Update)
	assert.EqualError(t, err, "model must not be a pointer")
}

func TestOutputDbPersistWithNilModelReturnsError(t *testing.T) {
	output := &OutputDb{}
	err := output.Persist(t.Context(), nil, db_repo.Update)
	assert.EqualError(t, err, "model must not be nil")
}
