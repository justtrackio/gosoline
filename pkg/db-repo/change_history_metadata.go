package db_repo

import (
	"bytes"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"github.com/justtrackio/gosoline/pkg/funk"
	"golang.org/x/exp/slices"
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
	scope     *gorm.DB
	tableName string
	fields    []*schema.Field
	relations map[string]*schema.Relationship
}

func (m *tableMetadataBuilder) build() (*tableMetadata, error) {
	metadata := &tableMetadata{}
	metadata.exists = m.scope.Migrator().HasTable(m.tableName)
	metadata.tableName = m.tableName

	buf := bytes.NewBufferString(metadata.tableNameQuoted)
	m.scope.QuoteTo(buf, m.tableName)
	metadata.tableNameQuoted = buf.String()

	var err error
	metadata.columns, err = m.buildColumns()
	if err != nil {
		return nil, err
	}

	metadata.primaryKeys, err = m.buildPrimaryKeys()
	if err != nil {
		return nil, err
	}

	return metadata, nil
}

func (m *tableMetadataBuilder) buildColumns() ([]columnMetadata, error) {
	var columns []columnMetadata
	for _, field := range m.fields {
		if field.EmbeddedSchema != nil {
			continue
		}

		if _, exists := m.relations[field.Name]; exists {
			continue
		}

		cm, err := m.buildColumn(field)
		if err != nil {
			return nil, err
		}

		columns = append(columns, *cm)
	}

	return columns, nil
}

func (m *tableMetadataBuilder) buildPrimaryKeys() ([]columnMetadata, error) {
	var columns []columnMetadata
	for _, field := range m.fields {
		if field.PrimaryKey {
			cm, err := m.buildColumn(field)
			if err != nil {
				return nil, err
			}

			columns = append(columns, *cm)
		}
	}
	return columns, nil
}

func (m *tableMetadataBuilder) buildColumn(field *schema.Field) (*columnMetadata, error) {
	name := field.DBName

	nameQuoted := ""
	nameBuf := bytes.NewBufferString(nameQuoted)
	m.scope.QuoteTo(nameBuf, field.DBName)
	nameQuoted = nameBuf.String()

	definition := ""
	definitionBuf := bytes.NewBufferString(definition)
	m.scope.QuoteTo(definitionBuf, field.DBName)
	definition = definitionBuf.String() + " " + m.dataTypeOfField(field)

	exists, err := m.hasColumn(m.tableName, field.DBName)
	if err != nil {
		return nil, fmt.Errorf("could not check if column exists: %w", err)
	}

	cm := m.getColumnMetadata(name, nameQuoted, definition, exists)

	return &cm, nil
}

func (m *tableMetadataBuilder) hasColumn(tableName, fieldName string) (bool, error) {
	database := m.scope.Migrator().CurrentDatabase()

	sql := "SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?"

	db := m.scope.Raw(sql, database, tableName, fieldName)
	if db.Error != nil {
		return false, db.Error
	}

	rows, err := db.Rows()
	if err != nil {
		return false, err
	}

	// close rows otherwise we're leaking connections
	defer rows.Close()

	if !rows.Next() {
		if rows.Err() != nil {
			return false, rows.Err()
		}

		return false, nil
	}

	var count int
	if err := rows.Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *tableMetadataBuilder) getColumnMetadata(name string, nameQuoted string, definition string, exists bool) columnMetadata {
	return columnMetadata{
		name:       name,
		nameQuoted: nameQuoted,
		definition: definition,
		exists:     exists,
	}
}

func (m *tableMetadataBuilder) dataTypeOfField(field *schema.Field) string {
	tag := m.scope.Migrator().FullDataTypeOf(field)
	sql := tag.SQL

	sql = strings.Replace(sql, "AUTO_INCREMENT", "", -1)
	sql = strings.Replace(sql, "UNIQUE", "", -1)

	return sql
}

func newTableMetadata(scope *gorm.DB, tableName string, fields []*schema.Field, relations map[string]*schema.Relationship) (*tableMetadata, error) {
	builder := tableMetadataBuilder{
		tableName: tableName,
		scope:     scope,
		fields:    fields,
		relations: relations,
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
	})
}

func (m *tableMetadata) definitions(items []columnMetadata) []string {
	return funk.Map(items, func(item columnMetadata) string {
		return item.definition
	})
}

func (m *tableMetadata) columnNamesQuotedExcludingValue(excluded ...string) []string {
	return m.namesQuoted(funk.Filter(m.columns, func(item columnMetadata) bool {
		return !slices.Contains(excluded, item.name)
	}))
}
