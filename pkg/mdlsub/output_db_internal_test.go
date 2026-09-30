package mdlsub

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jinzhu/gorm"
	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/exec"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/require"
)

func TestOutputDbOrmCloseKeepsSharedConnectionOpen(t *testing.T) {
	connection, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, connection.Close())
		require.NoError(t, mock.ExpectationsWereMet())
	})

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	client := db.NewClientWithInterfaces(logger, sqlx.NewDb(connection, "mysql"), exec.NewDefaultExecutor())
	orm, err := gorm.Open("mysql", outputDbOrmClient{client: client})
	require.NoError(t, err)

	_ = orm.Close()

	mock.ExpectQuery("SELECT 1").WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow(1))
	rows, err := client.Query(t.Context(), "SELECT 1")
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next())
	var value int
	require.NoError(t, rows.Scan(&value))
	require.Equal(t, 1, value)
}
