package sql_test

import (
	"github.com/applike/gosoline/pkg/apiserver/sql"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/stretchr/testify/assert"
	"testing"
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
					Values:    []interface{}{"blub"},
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
					Values:    []interface{}{"blub"},
				},
			},
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
	expected.GroupBy("id")
	expected.OrderBy("fieldA", "ASC")
	expected.OrderBy("fieldB", "DESC")
	expected.Page(0, 3)

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
					Values:    []interface{}{"blub", "blubber"},
				},
				{
					Dimension: "fieldA",
					Operator:  "!=",
					Values:    []interface{}{1},
				},
				{
					Dimension: "void",
					Operator:  "is",
					Values:    []interface{}{"null"},
				},
			},
			Groups: []sql.Filter{
				{
					Matches: []sql.FilterMatch{
						{
							Dimension: "fieldB",
							Operator:  "~",
							Values:    []interface{}{"foo"},
						},
						{
							Dimension: "fieldB",
							Operator:  "=",
							Values:    []interface{}{"bar"},
						},
					},
					Bool: "or",
				},
			},
			Bool: "and",
		},
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	assert.NoError(t, err)

	expected := db_repo.NewQueryBuilder()
	expected.Table("tablename")
	expected.Where(
		"(((foo  IN (?,?))) and ((fieldA != ?)) and ((void IS null)) and (((fieldB LIKE ?)) or ((fieldB = ?))))",
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
		},
	}

	inp := &sql.Input{
		Filter: sql.Filter{
			Matches: []sql.FilterMatch{
				{
					Dimension: "sql1",
					Operator:  "=",
					Values:    []interface{}{"value1", nil},
				},
				{
					Dimension: "go1",
					Operator:  "=",
					Values:    []interface{}{"value2", nil},
				},
				{
					Dimension: "sql2",
					Operator:  "=",
					Values:    []interface{}{nil},
				},
				{
					Dimension: "go2",
					Operator:  "=",
					Values:    []interface{}{nil},
				},
				{
					Dimension: "sql3",
					Operator:  "!=",
					Values:    []interface{}{"value3", nil},
				},
				{
					Dimension: "go3",
					Operator:  "!=",
					Values:    []interface{}{"value4", nil},
				},
				{
					Dimension: "sql4",
					Operator:  "!=",
					Values:    []interface{}{nil},
				},
				{
					Dimension: "go4",
					Operator:  "!=",
					Values:    []interface{}{nil},
				},
			},
			Groups: []sql.Filter{},
			Bool:   "and",
		},
	}

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	assert.NoError(t, err)

	expected := db_repo.NewQueryBuilder()
	expected.Table("tablename")
	expected.Where(
		"(((sql1  IN (?,?))) and ((go1  IN (?,?) OR go1 IS  NULL)) and ((sql2 = ?)) and ((go2 IS NULL)) and ((sql3 NOT IN (?,?))) and ((go3 NOT IN (?) AND go3 IS NOT NULL)) and ((sql4 != ?)) and ((go4 IS NOT NULL)))",
		"value1",
		nil,
		"value2",
		nil,
		nil,
		"value3",
		nil,
		"value4",
		nil,
	)
	expected.GroupBy("id")

	assert.Equal(t, expected, qb)
}
