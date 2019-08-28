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
			"id":     db_repo.NewSimpleFieldMapping("id"),
			"bla":    db_repo.NewSimpleFieldMapping("foo"),
			"fieldA": db_repo.NewSimpleFieldMapping("fieldA"),
			"fieldB": db_repo.NewSimpleFieldMapping("fieldB"),
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
			"id":     db_repo.NewSimpleFieldMapping("id"),
			"bla":    db_repo.NewSimpleFieldMapping("foo"),
			"void":   db_repo.NewSimpleFieldMapping("void"),
			"fieldA": db_repo.NewSimpleFieldMapping("fieldA"),
			"fieldB": db_repo.NewSimpleFieldMapping("fieldB"),
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
	expected.Where("(((foo IN (?,?))) and ((fieldA != ?)) and ((void IS null)) and (((fieldB LIKE ?)) or ((fieldB = ?))))", "blub", "blubber", 1, "%foo%", "bar")
	expected.GroupBy("id")

	assert.Equal(t, expected, qb)
}
