package assert

import (
	"database/sql"
	"fmt"
	test2 "github.com/applike/gosoline/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func SqlTableHasOneRowOnly(t *testing.T, tableName string) {
	rows := test2.IntegrationTestDb.QueryRow(fmt.Sprintf("select count(*) from %s", tableName))
	count := "count(*)"
	err := rows.Scan(&count)

	if err != nil && err != sql.ErrNoRows {
		assert.Fail(t, "error retrieving count from database", err.Error())
		return
	}

	if err == sql.ErrNoRows {
		assert.Fail(t, "table has 0 rows", err.Error())
		return
	}

	assert.Equal(t, "1", count, "there is more than 1 row in the table")
}

func SqlColumnHasSpecificValue(t *testing.T, tableName string, column string, expectedValue interface{}) {
	query := fmt.Sprintf("select %s from %s where %s = '%v'", column, tableName, column, expectedValue)
	row := test2.IntegrationTestDb.QueryRow(query)
	err := row.Scan(&column)

	if err != nil && err != sql.ErrNoRows {
		assert.Fail(t, "error querying database", err.Error())
		return
	}

	if err == sql.ErrNoRows {
		assert.Fail(t, fmt.Sprintf("no rows existing with expected value: %v in column: %s", expectedValue, column))
		return
	}

	assert.Equal(t, expectedValue, column)
}
