package assert

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type rowQuerying interface {
	QueryRow(query string, args ...any) *sql.Row
}

func SqlTableHasOneRowOnly(t *testing.T, db rowQuerying, tableName string) {
	rows := db.QueryRow(fmt.Sprintf("select count(*) from %s", tableName))
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

func SqlColumnHasSpecificValue(t *testing.T, db rowQuerying, tableName string, column string, expectedValue any) {
	query := fmt.Sprintf("select %s from %s where %s = '%v'", column, tableName, column, expectedValue)
	row := db.QueryRow(query)
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
