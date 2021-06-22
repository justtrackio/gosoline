package db_test

import (
	goSqlMock "github.com/DATA-DOG/go-sqlmock"
	"github.com/applike/gosoline/pkg/db"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetResult(t *testing.T) {
	client, sqlMock := getMocks()

	expectedResult := db.Result{
		{
			"id":   "3",
			"name": "foo",
		},
		{
			"id":   "2",
			"name": "bar",
		},
	}

	rows := goSqlMock.NewRows([]string{"id", "name"})
	rows.AddRow("3", "foo")
	rows.AddRow("2", "bar")

	sqlMock.ExpectQuery("^SELECT (.+) FROM TestTable").WillReturnRows(rows)

	result, err := client.GetResult("SELECT * FROM TestTable;")

	if !assert.Nil(t, err) {
		return
	}

	if !assert.Len(t, *result, len(expectedResult)) {
		return
	}

	for i, row := range *result {
		assert.Equal(t, expectedResult[i]["id"], row["id"])
		assert.Equal(t, expectedResult[i]["name"], row["name"])
	}
}

func TestGetSingleScalarValue(t *testing.T) {
	client, sqlMock := getMocks()

	rows := goSqlMock.NewRows([]string{"count"})
	rows.AddRow(3)

	sqlMock.ExpectQuery("^SELECT (.+) AS count FROM TestTable").WillReturnRows(rows)

	count, err := client.GetSingleScalarValue("SELECT COUNT(id) AS count FROM TestTable")

	if !assert.Nil(t, err) {
		return
	}

	assert.Equal(t, 3, count)
}

func TestQuery(t *testing.T) {
	client, sqlMock := getMocks()

	id := "1"
	name := "test_thing"
	campaignId := "3"
	revenueEur := "60"

	columns := []string{"id", "name", "campaignId", "revenueEur"}
	rows := goSqlMock.NewRows(columns)
	rows.AddRow(id, name, campaignId, revenueEur)

	sqlMock.ExpectQuery("^SELECT (.+) FROM TestTable").WillReturnRows(rows)

	sqlRows, err := client.Query("SELECT * FROM TestTable;")
	assert.Nil(t, err)

	var resultId string
	var resultName string
	var resultCampaignId string
	var resultRevenueEur string

	sqlRows.Next()
	err = sqlRows.Scan(&resultId, &resultName, &resultCampaignId, &resultRevenueEur)
	assert.Nil(t, err)

	assert.Equal(t, id, resultId)
	assert.Equal(t, name, resultName)
	assert.Equal(t, campaignId, resultCampaignId)
	assert.Equal(t, revenueEur, resultRevenueEur)

	sqlMock.ExpectClose()
}

func TestClient_Exec(t *testing.T) {
	client, sqlMock := getMocks()

	id := "2"
	name := "old_name"
	newName := "new_name"

	columns := []string{"id", "name"}
	rows := goSqlMock.NewRows(columns)
	rows.AddRow(id, name)

	sqlMock.ExpectExec("UPDATE Campaign").WithArgs(newName, id).WillReturnResult(goSqlMock.NewResult(0, 1))

	result, err := client.Exec("UPDATE Campaign SET name = ? WHERE id = ?", newName, id)
	assert.Nil(t, err)

	rowsAffected, err := result.RowsAffected()
	assert.Nil(t, err)

	assert.Equal(t, int64(1), rowsAffected)

	sqlMock.ExpectClose()
}

func getMocks() (db.Client, goSqlMock.Sqlmock) {
	dbMock, sqlMock, _ := goSqlMock.New()
	loggerMock := logMocks.NewLoggerMockedAll()
	sqlxDB := sqlx.NewDb(dbMock, "sqlmock")

	client := db.NewClientWithInterfaces(loggerMock, sqlxDB)

	return client, sqlMock
}
