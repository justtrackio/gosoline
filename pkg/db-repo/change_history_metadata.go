package db_repo

import (
	"github.com/jinzhu/gorm"
	"github.com/thoas/go-funk"
	"strings"
)

type columnMetadata struct {
	exists     bool
	name       string
	nameQuoted string
	definition string
}

type tableMetadata struct {
	exists          bool
	tableName       string
	tableNameQuoted string
	columns         []columnMetadata
	primaryKeys     []columnMetadata
}

type tableMetadataBuilder struct {
	scope     *gorm.Scope
	tableName string
	fields    []*gorm.StructField
}

func (m *tableMetadataBuilder) build() *tableMetadata {
	metadata := &tableMetadata{}
	metadata.exists = m.scope.Dialect().HasTable(m.tableName)
	metadata.tableName = m.tableName
	metadata.tableNameQuoted = m.scope.Quote(m.tableName)
	metadata.columns = m.buildColumns()
	metadata.primaryKeys = m.buildPrimaryKeys()
	return metadata
}

func (m *tableMetadataBuilder) buildColumns() []columnMetadata {
	var columns []columnMetadata
	for _, field := range m.fields {
		if field.IsNormal {
			columns = append(columns, m.buildColumn(field))
		}
	}
	return columns
}

func (m *tableMetadataBuilder) buildPrimaryKeys() []columnMetadata {
	var columns []columnMetadata
	for _, field := range m.fields {
		if field.IsPrimaryKey {
			columns = append(columns, m.buildColumn(field))
		}
	}
	return columns
}

func (m *tableMetadataBuilder) buildColumn(field *gorm.StructField) (cm columnMetadata) {
	name := field.DBName
	nameQuoted := m.scope.Quote(field.DBName)
	definition := m.scope.Quote(field.DBName) + " " + m.dataTypeOfField(field)

	defer func() {
		err := recover()
		if err != nil {
			cm = m.getColumnMetadata(name, nameQuoted, definition, false)
		}
	}()

	exists := m.scope.Dialect().HasColumn(m.tableName, field.DBName)
	cm = m.getColumnMetadata(name, nameQuoted, definition, exists)

	return
}

func (m *tableMetadataBuilder) getColumnMetadata(name string, nameQuoted string, definition string, exists bool) columnMetadata {
	return columnMetadata{
		name:       name,
		nameQuoted: nameQuoted,
		definition: definition,
		exists:     exists,
	}
}

func (m *tableMetadataBuilder) dataTypeOfField(field *gorm.StructField) string {
	tag := m.scope.Dialect().DataTypeOf(field)

	tag = strings.Replace(tag, "AUTO_INCREMENT", "", -1)
	tag = strings.Replace(tag, "UNIQUE", "", -1)

	return tag
}

func newTableMetadata(scope *gorm.Scope, tableName string, fields []*gorm.StructField) *tableMetadata {
	builder := tableMetadataBuilder{
		tableName: tableName,
		scope:     scope,
		fields:    fields,
	}
	return builder.build()
}

func (m *tableMetadata) columnNamesQuoted() []string {
	return m.namesQuoted(m.columns)
}

func (m *tableMetadata) primaryKeyNamesQuoted() []string {
	return m.namesQuoted(m.primaryKeys)
}

func (m *tableMetadata) columnDefinitions() []string {
	return m.definitions(m.columns)
}

func (m *tableMetadata) namesQuoted(items []columnMetadata) []string {
	return funk.Map(items, func(item columnMetadata) string {
		return item.nameQuoted
	}).([]string)
}

func (m *tableMetadata) definitions(items []columnMetadata) []string {
	return funk.Map(items, func(item columnMetadata) string {
		return item.definition
	}).([]string)
}

func (m *tableMetadata) columnNamesQuotedExcludingValue(excluded ...string) []string {
	return m.namesQuoted(funk.Filter(m.columns, func(item columnMetadata) bool {
		return !funk.ContainsString(excluded, item.name)
	}).([]columnMetadata))
}
