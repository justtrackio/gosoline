package sql_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver/sql"
	"github.com/stretchr/testify/assert"
)

func TestListQueryBuilder_Build_TableNameMissing(t *testing.T) {
	metadata := db_repo.Metadata{}
	inp := &sql.Input{}

	lqb := sql.NewOrmQueryBuilder(metadata)
	_, err := lqb.Build(inp)

	assert.EqualError(t, err, "no table name defined")
}

func TestListQueryBuilder_Build_IdMissing(t *testing.T) {
	metadata := db_repo.Metadata{
		TableName: "tablename",
	}
	inp := &sql.Input{}

	lqb := sql.NewOrmQueryBuilder(metadata)
	_, err := lqb.Build(inp)

	assert.EqualError(t, err, "no primary key defined")
}

func TestListQueryBuilder_Build_DimensionMissing(t *testing.T) {
	metadata := db_repo.Metadata{}
	inp := &sql.Input{
		Filter: sql.Filter{
			Matches: []sql.FilterMatch{
				{
					Dimension: "bla",
					Operator:  "=",
					Values:    []any{"blub"},
				},
			},
		},
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	_, err := lqb.Build(inp)

	assert.EqualError(t, err, "no list mapping found for dimension bla")
}

func TestListQueryBuilder_Build(t *testing.T) {
	metadata := db_repo.Metadata{
		TableName:  "tablename",
		PrimaryKey: "id",
		Mappings: db_repo.FieldMappings{
			"id":     db_repo.NewFieldMapping("id"),
			"bla":    db_repo.NewFieldMapping("foo"),
			"fieldA": db_repo.NewFieldMapping("fieldA"),
			"fieldB": db_repo.NewFieldMapping("fieldB"),
		},
	}

	inp := &sql.Input{
		Filter: sql.Filter{
			Matches: []sql.FilterMatch{
				{
					Dimension: "bla",
					Operator:  "=",
					Values:    []any{"blub"},
				},
			},
			Bool: sql.BoolAnd,
		},
		Order: []sql.Order{
			{
				Field:     "fieldA",
				Direction: "ASC",
			},
			{
				Field:     "fieldB",
				Direction: "DESC",
			},
		},
		GroupBy: []string{"bla"},
		Page: &sql.Page{
			Offset: 0,
			Limit:  3,
		},
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	assert.NoError(t, err)

	expected := db_repo.NewQueryBuilder()
	expected.Table("tablename")
	expected.Where("(((foo = ?)))", "blub")
	expected.GroupBy("id", "foo")
	expected.OrderBy("fieldA", "ASC")
	expected.OrderBy("fieldB", "DESC")
	expected.Page(0, 3)

	assert.Equal(t, expected, qb)
}

func TestListQueryBuilder_BuildWithJoin(t *testing.T) {
	metadata := db_repo.Metadata{
		TableName:  "tablename",
		PrimaryKey: "id",
		Mappings: db_repo.FieldMappings{
			"id":     db_repo.NewFieldMapping("id"),
			"bla":    db_repo.NewFieldMapping("foo").WithJoin("JOIN footable"),
			"fieldA": db_repo.NewFieldMapping("fieldA"),
			"fieldB": db_repo.NewFieldMapping("fieldB").WithJoin("JOIN fieldBTable"),
		},
	}

	inp := &sql.Input{
		Filter: sql.Filter{
			Matches: []sql.FilterMatch{
				{
					Dimension: "bla",
					Operator:  "=",
					Values:    []any{"blub"},
				},
			},
			Bool: sql.BoolAnd,
		},
		GroupBy: []string{"fieldB"},
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	assert.NoError(t, err)

	expected := db_repo.NewQueryBuilder()
	expected.Table("tablename")
	expected.Where("(((foo = ?)))", "blub")
	expected.GroupBy("id", "fieldB")
	expected.Joins([]string{
		"JOIN footable",
		"JOIN fieldBTable",
	})

	assert.Equal(t, expected, qb)
}

func TestListQueryBuilder_Build_ComplexFilter(t *testing.T) {
	metadata := db_repo.Metadata{
		PrimaryKey: "id",
		TableName:  "tablename",
		Mappings: db_repo.FieldMappings{
			"id":     db_repo.NewFieldMapping("id"),
			"bla":    db_repo.NewFieldMapping("foo"),
			"void":   db_repo.NewFieldMapping("void"),
			"fieldA": db_repo.NewFieldMapping("fieldA"),
			"fieldB": db_repo.NewFieldMapping("fieldB"),
		},
	}

	inp := &sql.Input{
		Filter: sql.Filter{
			Matches: []sql.FilterMatch{
				{
					Dimension: "bla",
					Operator:  "=",
					Values:    []any{"blub", "blubber"},
				},
				{
					Dimension: "fieldA",
					Operator:  "!=",
					Values:    []any{1},
				},
				{
					Dimension: "void",
					Operator:  "is",
					Values:    []any{"null"},
				},
			},
			Groups: []sql.Filter{
				{
					Matches: []sql.FilterMatch{
						{
							Dimension: "fieldB",
							Operator:  "~",
							Values:    []any{"foo"},
						},
						{
							Dimension: "fieldB",
							Operator:  "=",
							Values:    []any{"bar"},
						},
					},
					Bool: sql.BoolOr,
				},
			},
			Bool: sql.BoolAnd,
		},
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	assert.NoError(t, err)

	expected := db_repo.NewQueryBuilder()
	expected.Table("tablename")
	expected.Where(
		"(((foo IN (?,?))) AND ((fieldA != ?)) AND ((void IS null)) AND (((fieldB LIKE ?)) OR ((fieldB = ?))))",
		"blub",
		"blubber",
		1,
		"%foo%",
		"bar",
	)
	expected.GroupBy("id")

	assert.Equal(t, expected, qb)
}

func TestListQueryBuilder_Build_NullFilter(t *testing.T) {
	metadata := db_repo.Metadata{
		PrimaryKey: "id",
		TableName:  "tablename",
		Mappings: db_repo.FieldMappings{
			"id":   db_repo.NewFieldMapping("id"),
			"sql1": db_repo.NewFieldMappingWithMode("sql1", db_repo.NullModeDefault),
			"go1":  db_repo.NewFieldMappingWithMode("go1", db_repo.NullModeDistinct),
			"sql2": db_repo.NewFieldMappingWithMode("sql2", db_repo.NullModeDefault),
			"go2":  db_repo.NewFieldMappingWithMode("go2", db_repo.NullModeDistinct),
			"sql3": db_repo.NewFieldMappingWithMode("sql3", db_repo.NullModeDefault),
			"go3":  db_repo.NewFieldMappingWithMode("go3", db_repo.NullModeDistinct),
			"sql4": db_repo.NewFieldMappingWithMode("sql4", db_repo.NullModeDefault),
			"go4":  db_repo.NewFieldMappingWithMode("go4", db_repo.NullModeDistinct),
			"sql5": db_repo.NewFieldMappingWithMode("sql5", db_repo.NullModeDefault),
			"go5":  db_repo.NewFieldMappingWithMode("go5", db_repo.NullModeDistinct),
			"sql6": db_repo.NewFieldMappingWithMode("sql6", db_repo.NullModeDefault),
			"go6":  db_repo.NewFieldMappingWithMode("go6", db_repo.NullModeDistinct),
		},
	}

	inp := &sql.Input{
		Filter: sql.Filter{
			Matches: []sql.FilterMatch{
				{
					Dimension: "sql1",
					Operator:  "=",
					Values:    []any{"value1", nil},
				},
				{
					Dimension: "go1",
					Operator:  "=",
					Values:    []any{"value2", nil},
				},
				{
					Dimension: "sql2",
					Operator:  "=",
					Values:    []any{nil},
				},
				{
					Dimension: "go2",
					Operator:  "=",
					Values:    []any{nil},
				},
				{
					Dimension: "sql3",
					Operator:  "!=",
					Values:    []any{"value3", nil},
				},
				{
					Dimension: "go3",
					Operator:  "!=",
					Values:    []any{"value4", nil},
				},
				{
					Dimension: "sql4",
					Operator:  "!=",
					Values:    []any{nil},
				},
				{
					Dimension: "go4",
					Operator:  "!=",
					Values:    []any{nil},
				},
				{
					Dimension: "sql5",
					Operator:  "=",
					Values:    []any{"value5", "value6"},
				},
				{
					Dimension: "go5",
					Operator:  "=",
					Values:    []any{"value7", "value8"},
				},
				{
					Dimension: "sql6",
					Operator:  "!=",
					Values:    []any{"value9", "value10"},
				},
				{
					Dimension: "go6",
					Operator:  "!=",
					Values:    []any{"value11", "value12"},
				},
			},
			Groups: []sql.Filter{},
			Bool:   sql.BoolAnd,
		},
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	assert.NoError(t, err)

	expected := db_repo.NewQueryBuilder()
	expected.Table("tablename")
	expected.Where(
		"(((sql1 IN (?,?))) AND ((go1 IN (?,?) OR go1 IS NULL)) AND ((sql2 = ?)) AND ((go2 IS NULL)) AND ((sql3 NOT IN (?,?))) AND "+
			"((go3 NOT IN (?) AND go3 IS NOT NULL)) AND ((sql4 != ?)) AND ((go4 IS NOT NULL)) AND ((sql5 IN (?,?))) AND ((go5 IN (?,?))) AND "+
			"((sql6 NOT IN (?,?))) AND ((go6 NOT IN (?,?) OR go6 IS NULL)))",
		"value1",
		nil,
		"value2",
		nil,
		nil,
		"value3",
		nil,
		"value4",
		nil,
		"value5",
		"value6",
		"value7",
		"value8",
		"value9",
		"value10",
		"value11",
		"value12",
	)
	expected.GroupBy("id")

	assert.Equal(t, expected, qb)
}

func TestListQueryBuilder_BuildSqlInjectionBool(t *testing.T) {
	metadata := db_repo.Metadata{
		TableName:  "tablename",
		PrimaryKey: "id",
		Mappings: db_repo.FieldMappings{
			"id":  db_repo.NewFieldMapping("id"),
			"bla": db_repo.NewFieldMapping("foo"),
		},
	}

	inp := &sql.Input{
		Filter: sql.Filter{
			Matches: []sql.FilterMatch{
				{
					Dimension: "bla",
					Operator:  "=",
					Values:    []any{"bla"},
				},
				{
					Dimension: "bla",
					Operator:  "=",
					Values:    []any{"bla"},
				},
			},
			Bool: "AND (SELECT password FROM admins where id = 42) = 'hunter2' AND",
		},
		Order:   []sql.Order{},
		GroupBy: []string{},
		Page:    nil,
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	assert.EqualError(
		t,
		err,
		`can not build filter: invalid boolean: "AND (SELECT password FROM admins where id = 42) = 'hunter2' AND", should be either "AND" or "OR"`,
	)
	assert.Equal(t, db_repo.NewQueryBuilder(), qb)
}

func TestListQueryBuilder_BuildSqlInjectionNestedBool(t *testing.T) {
	metadata := db_repo.Metadata{
		TableName:  "tablename",
		PrimaryKey: "id",
		Mappings: db_repo.FieldMappings{
			"id":  db_repo.NewFieldMapping("id"),
			"bla": db_repo.NewFieldMapping("foo"),
		},
	}

	inp := &sql.Input{
		Filter: sql.Filter{
			Groups: []sql.Filter{
				{
					Matches: []sql.FilterMatch{
						{
							Dimension: "bla",
							Operator:  "=",
							Values:    []any{"bla"},
						},
						{
							Dimension: "bla",
							Operator:  "=",
							Values:    []any{"bla"},
						},
					},
					Groups: []sql.Filter{},
					Bool:   "AND (SELECT password FROM admins where id = 42) = 'hunter2' AND",
				},
			},
			Bool: sql.BoolAnd,
		},
		Order:   []sql.Order{},
		GroupBy: []string{},
		Page:    nil,
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	assert.EqualError(
		t,
		err,
		`can not build filter: invalid boolean: "AND (SELECT password FROM admins where id = 42) = 'hunter2' AND", should be either "AND" or "OR"`,
	)
	assert.Equal(t, db_repo.NewQueryBuilder(), qb)
}

func TestListQueryBuilder_BuildSqlInjectionOperator(t *testing.T) {
	metadata := db_repo.Metadata{
		TableName:  "tablename",
		PrimaryKey: "id",
		Mappings: db_repo.FieldMappings{
			"id":  db_repo.NewFieldMapping("id"),
			"bla": db_repo.NewFieldMapping("foo"),
		},
	}

	inp := &sql.Input{
		Filter: sql.Filter{
			Matches: []sql.FilterMatch{
				{
					Dimension: "bla",
					Operator:  "IN (SELECT username FROM admins WHERE id = 42) AND 1 =",
					Values:    []any{1},
				},
			},
			Bool: sql.BoolAnd,
		},
		Order:   []sql.Order{},
		GroupBy: []string{},
		Page:    nil,
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	assert.EqualError(
		t,
		err,
		`can not build filter: error building filter for column foo: invalid operator "IN (SELECT username FROM admins WHERE id = 42) AND 1 ="`,
	)
	assert.Equal(t, db_repo.NewQueryBuilder(), qb)
}
