package db_repo

import "github.com/justtrackio/gosoline/pkg/mdl"

const (
	BoolAnd = "AND"
	BoolOr  = "OR"

	// Nulls will be treated like SQL defines them. null != 1 yields null
	NullModeDefault = 0
	// Nulls will be treated like Go defines them. null != 1 yields true
	NullModeDistinct = 1
)

type NullMode int

type Metadata struct {
	ModelId    mdl.ModelId
	TableName  string
	PrimaryKey string
	Mappings   FieldMappings
}

type FieldMappings map[string]FieldMapping

type FieldMappingColumn struct {
	name     string
	nullMode NullMode
}

func (c FieldMappingColumn) Name() string {
	return c.name
}

func (c FieldMappingColumn) NullMode() NullMode {
	return c.nullMode
}

type FieldMapping struct {
	columns []FieldMappingColumn
	joins   []string
	bool    string
}

func (f FieldMapping) Columns() []FieldMappingColumn {
	return f.columns
}

func (f FieldMapping) Joins() []string {
	return f.joins
}

func (f FieldMapping) Bool() string {
	return f.bool
}

func (f FieldMapping) ColumnNames() []string {
	names := make([]string, 0, len(f.columns))

	for _, c := range f.columns {
		names = append(names, c.name)
	}

	return names
}

func NewFieldMapping(column string) FieldMapping {
	return NewFieldMappingWithMode(column, NullModeDefault)
}

func NewFieldMappingWithMode(column string, nullMode NullMode) FieldMapping {
	return FieldMapping{
		columns: []FieldMappingColumn{
			{
				name:     column,
				nullMode: nullMode,
			},
		},
		bool: BoolOr,
	}
}

func (f FieldMapping) WithColumn(column string) FieldMapping {
	return f.WithColumnWithMode(column, NullModeDefault)
}

func (f FieldMapping) WithColumnWithMode(column string, nullMode NullMode) FieldMapping {
	f.columns = append(f.columns, FieldMappingColumn{
		name:     column,
		nullMode: nullMode,
	})

	return f
}

func (f FieldMapping) WithJoin(join string) FieldMapping {
	f.joins = append(f.joins, join)

	return f
}

func (f FieldMapping) WithBool(boolOp string) FieldMapping {
	f.bool = boolOp

	return f
}
