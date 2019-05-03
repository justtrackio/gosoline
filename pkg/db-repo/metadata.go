package db_repo

import "github.com/applike/gosoline/pkg/mdl"

const (
	BoolAnd = "AND"
	BoolOr  = "OR"
)

type Metadata struct {
	ModelId    mdl.ModelId
	TableName  string
	PrimaryKey string
	Mappings   FieldMappings
}

type FieldMappings map[string]FieldMapping

type FieldMapping struct {
	Columns []string
	Joins   []string
	Bool    string
}

func NewSimpleFieldMapping(column string) FieldMapping {
	return FieldMapping{
		Columns: []string{column},
		Bool:    BoolOr,
	}
}

func NewJoinedFieldMapping(column string, join string) FieldMapping {
	return FieldMapping{
		Columns: []string{column},
		Joins:   []string{join},
		Bool:    BoolOr,
	}
}
