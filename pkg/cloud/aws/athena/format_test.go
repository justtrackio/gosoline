package athena_test

import (
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/athena"
	"github.com/stretchr/testify/assert"
)

func TestReplaceDollarPlaceholders(t *testing.T) {
	type unknownType struct {
		Id int
	}

	tests := map[string]struct {
		value  any
		expSql string
		expErr string
	}{
		"nil": {
			value:  nil,
			expSql: "SELECT column FROM table WHERE value IS NULL",
		},
		"bool": {
			value:  true,
			expSql: "SELECT column FROM table WHERE value = true",
		},
		"string": {
			value:  "foo",
			expSql: "SELECT column FROM table WHERE value = 'foo'",
		},
		"string_escape\\n": {
			value:  "foo\nbar",
			expSql: "SELECT column FROM table WHERE value = 'foo\\nbar'",
		},
		"string_escape\\r": {
			value:  "foo\rbar",
			expSql: "SELECT column FROM table WHERE value = 'foo\\rbar'",
		},
		"string_escape\\0": {
			value:  "foo\000bar",
			expSql: "SELECT column FROM table WHERE value = 'foo\\0bar'",
		},
		"string_escape\\Z": {
			value:  "foo\032bar",
			expSql: "SELECT column FROM table WHERE value = 'foo\\Zbar'",
		},
		"byte": {
			value:  []byte("foo"),
			expSql: "SELECT column FROM table WHERE value = 'foo'",
		},
		"int": {
			value:  42,
			expSql: "SELECT column FROM table WHERE value = 42",
		},
		"float": {
			value:  float64(42.24),
			expSql: "SELECT column FROM table WHERE value = 42.24",
		},
		"unknown": {
			value:  unknownType{3},
			expErr: "unsupported type athena_test.unknownType for arg[0]: {3}",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			qb := squirrel.Select("column").From("table").Where(squirrel.Eq{"value": tc.value})
			qry, args, err := qb.PlaceholderFormat(squirrel.Dollar).ToSql()
			assert.NoError(t, err)

			sql, err := athena.ReplaceDollarPlaceholders(qry, args)

			if tc.expErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tc.expErr)
			}

			assert.Equal(t, tc.expSql, sql)
		})
	}
}
