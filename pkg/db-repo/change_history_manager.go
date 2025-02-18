package db_repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type ChangeHistoryManagerSettings struct {
	TableSuffix      string `cfg:"table_suffix" default:"history"`
	MigrationEnabled bool   `cfg:"migration_enabled" default:"false"`
}

type ChangeHistoryManager struct {
	logger   log.Logger
	orm      *gorm.DB
	settings *ChangeHistoryManagerSettings
	models   []ModelBased
}

type changeHistoryManagerAppCtxKey int

func ProvideChangeHistoryManager(ctx context.Context, config cfg.Config, logger log.Logger) (*ChangeHistoryManager, error) {
	return appctx.Provide(ctx, changeHistoryManagerAppCtxKey(0), func() (*ChangeHistoryManager, error) {
		return NewChangeHistoryManager(ctx, config, logger)
	})
}

func NewChangeHistoryManager(ctx context.Context, config cfg.Config, logger log.Logger) (*ChangeHistoryManager, error) {
	orm, err := NewOrm(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	settings := &ChangeHistoryManagerSettings{}
	config.UnmarshalKey("change_history", settings)

	return NewChangeHistoryManagerWithInterfaces(logger, orm, settings), nil
}

func NewChangeHistoryManagerWithInterfaces(logger log.Logger, orm *gorm.DB, settings *ChangeHistoryManagerSettings) *ChangeHistoryManager {
	return &ChangeHistoryManager{
		logger:   logger.WithChannel("change_history_manager"),
		orm:      orm,
		settings: settings,
	}
}

func (c *ChangeHistoryManager) addModels(models ...ModelBased) {
	c.models = append(c.models, models...)
}

func (c *ChangeHistoryManager) RunMigrations() error {
	cfn := coffin.New()

	cfn.Go(func() error {
		for _, model := range funk.UniqByType(c.models) {
			cfn.Go(func() error {
				if err := c.RunMigration(model); err != nil {
					return fmt.Errorf("can not run migration for model %T: %w", model, err)
				}

				return nil
			})
		}

		return nil
	})

	return cfn.Wait()
}

func (c *ChangeHistoryManager) RunMigration(model ModelBased) error {
	originalTable := c.buildOriginalTableMetadata(model)
	historyTable := c.buildHistoryTableMetadata(model, originalTable)

	if err := c.executeMigration(originalTable, historyTable); err != nil {
		return fmt.Errorf("cannot execute change history migration: %w", err)
	}

	if err := c.validateSchema(originalTable.tableName, historyTable.tableName); err != nil {
		return fmt.Errorf("error during schema validation: %w", err)
	}

	return nil
}

func (c *ChangeHistoryManager) validateSchema(originalTable, historyTable string) error {
	qry := `
select original.TABLE_NAME, original.COLUMN_NAME, original.DATA_TYPE, history_entries.DATA_TYPE
from (select TABLE_NAME, COLUMN_NAME, DATA_TYPE
      from information_schema.columns
      where table_schema = '%s' and TABLE_NAME = '%s') original
         left join (select TABLE_NAME, COLUMN_NAME, DATA_TYPE
               from information_schema.columns
               where table_schema = '%s' and TABLE_NAME = '%s') as history_entries
              on original.COLUMN_NAME = history_entries.COLUMN_NAME
where original.DATA_TYPE != coalesce(history_entries.DATA_TYPE, '');`

	rows, err := c.getTableMetaData(4, func(database string) string {
		return fmt.Sprintf(qry, database, originalTable, database, historyTable)
	})
	if err != nil {
		return fmt.Errorf("cannot fetch table metadata: %w", err)
	}

	invalidColumnsErr := &multierror.Error{}

	for _, row := range rows {
		tableName := row[0]
		columnName := row[1]
		originalType := row[2]
		historyType := row[3]

		if historyType == "" {
			err = fmt.Errorf("missing column %s of type %s on history table %s", columnName, originalType, tableName)
		} else {
			err = fmt.Errorf("type mismatch for table %s and column %s: expected %s, got %s", tableName, columnName, originalType, historyType)
		}

		invalidColumnsErr = multierror.Append(invalidColumnsErr, err)
	}

	return invalidColumnsErr.ErrorOrNil()
}

func (c *ChangeHistoryManager) executeMigration(originalTable, historyTable *tableMetadata) error {
	statements := make([]string, 0)

	if !historyTable.exists {
		statements = append(statements, c.createHistoryTable(historyTable))
		statements = append(statements, c.dropHistoryTriggers(originalTable, historyTable)...)
		statements = append(statements, c.createHistoryTriggers(originalTable, historyTable)...)

		c.logger.Info("creating change history setup")

		return c.execute(statements)
	}

	tableUpdates, err := c.updateHistoryTable(originalTable, historyTable)
	if err != nil {
		return err
	}

	if len(tableUpdates) > 0 {
		statements = append(statements, tableUpdates...)
		statements = append(statements, c.dropHistoryTriggers(originalTable, historyTable)...)
		statements = append(statements, c.createHistoryTriggers(originalTable, historyTable)...)

		c.logger.Info("updating change history setup")

		return c.execute(statements)
	}

	c.logger.Info("change history setup was already up to date")

	return nil
}

func (c *ChangeHistoryManager) buildOriginalTableMetadata(model ModelBased) *tableMetadata {
	scope := c.orm.NewScope(model)
	fields := scope.GetModelStruct().StructFields
	tableName := scope.TableName()

	return newTableMetadata(scope, tableName, fields)
}

func (c *ChangeHistoryManager) buildHistoryTableMetadata(model ModelBased, originalTable *tableMetadata) *tableMetadata {
	historyScope := c.orm.NewScope(ChangeHistoryModel{})
	tableName := fmt.Sprintf("%s_%s", originalTable.tableName, c.settings.TableSuffix)
	modelFields := funk.Filter(c.orm.NewScope(model).GetModelStruct().StructFields, func(field *gorm.StructField) bool {
		// filter out history author id, it may be added twice; once by ChangeAuthorEmbeddable, once by HistoryEmbeddable
		return field.DBName != changeHistoryAuthorField
	})
	fields := append(historyScope.GetModelStruct().StructFields, modelFields...)

	return newTableMetadata(historyScope, tableName, fields)
}

func (c *ChangeHistoryManager) createHistoryTable(historyTable *tableMetadata) string {
	return fmt.Sprintf("CREATE TABLE %v (%v, PRIMARY KEY (%v))",
		historyTable.tableNameQuoted,
		strings.Join(historyTable.columnDefinitions(), ","),
		strings.Join(historyTable.primaryKeyNamesQuoted(), ","),
	)
}

func (c *ChangeHistoryManager) dropHistoryTriggers(originalTable *tableMetadata, historyTable *tableMetadata) []string {
	statements := make([]string, 0)
	triggers := []string{
		originalTable.tableName + "_ai",
		originalTable.tableName + "_au",
		originalTable.tableName + "_bd",
		historyTable.tableName + "_revai",
	}

	for _, trigger := range triggers {
		statements = append(statements, fmt.Sprintf(`DROP TRIGGER IF EXISTS %s`, trigger))
	}

	return statements
}

func (c *ChangeHistoryManager) createHistoryTriggers(originalTable *tableMetadata, historyTable *tableMetadata) []string {
	const NewRecord = "NEW"
	const OldRecord = "OLD"

	statements := []string{
		fmt.Sprintf(`CREATE TRIGGER %s_ai AFTER INSERT ON %s FOR EACH ROW %s WHERE %s`,
			originalTable.tableName,
			originalTable.tableNameQuoted,
			c.insertHistoryEntry(originalTable, historyTable, "insert"),
			c.primaryKeysMatchCondition(originalTable, NewRecord),
		),
		fmt.Sprintf(`CREATE TRIGGER %s_au AFTER UPDATE ON %s FOR EACH ROW %s WHERE %s AND (%s)`,
			originalTable.tableName,
			originalTable.tableNameQuoted,
			c.insertHistoryEntry(originalTable, historyTable, "update"),
			c.primaryKeysMatchCondition(originalTable, NewRecord),
			c.rowUpdatedCondition(originalTable),
		),
		fmt.Sprintf(`CREATE TRIGGER %s_bd BEFORE DELETE ON %s FOR EACH ROW %s WHERE %s`,
			originalTable.tableName,
			originalTable.tableNameQuoted,
			c.insertHistoryEntry(originalTable, historyTable, "delete"),
			c.primaryKeysMatchCondition(originalTable, OldRecord),
		),
		fmt.Sprintf(`CREATE TRIGGER %s_revai BEFORE INSERT ON %s FOR EACH ROW %s`,
			historyTable.tableName,
			historyTable.tableNameQuoted,
			c.incrementRevision(originalTable, historyTable),
		),
	}

	return statements
}

func (c *ChangeHistoryManager) insertHistoryEntry(originalTable *tableMetadata, historyTable *tableMetadata, action string) string {
	columnNames := originalTable.columnNamesQuotedExcludingValue(changeHistoryAuthorField)

	columns := strings.Join(columnNames, ",") + ",`change_history_author_id`"
	values := "d." + strings.Join(columnNames, ", d.") + ", @change_history_author_id"

	return fmt.Sprintf(`
		INSERT INTO %s (change_history_action,change_history_revision,change_history_action_at,%s) 
			SELECT '%s', NULL, NOW(), %s 
			FROM %s AS d`,
		historyTable.tableNameQuoted,
		columns,
		action,
		values,
		originalTable.tableNameQuoted)
}

func (c *ChangeHistoryManager) incrementRevision(originalTable *tableMetadata, historyTable *tableMetadata) string {
	return fmt.Sprintf(`SET NEW.change_history_revision = (SELECT IFNULL(MAX(d.change_history_revision), 0) + 1 FROM %s as d WHERE %s);`,
		historyTable.tableNameQuoted,
		c.primaryKeysMatchCondition(originalTable, "NEW"),
	)
}

func (c *ChangeHistoryManager) primaryKeysMatchCondition(originalTable *tableMetadata, record string) string {
	var conditions []string
	for _, columnName := range originalTable.primaryKeyNamesQuoted() {
		condition := fmt.Sprintf("d.%s = %s.%s", columnName, record, columnName)
		conditions = append(conditions, condition)
	}
	return strings.Join(conditions, " AND ")
}

func (c *ChangeHistoryManager) rowUpdatedCondition(originalTable *tableMetadata) string {
	columnNames := originalTable.columnNamesQuotedExcludingValue(changeHistoryAuthorField, ColumnUpdatedAt)
	var conditions []string
	for _, columnName := range columnNames {
		condition := fmt.Sprintf("NOT (OLD.%s <=> NEW.%s)", columnName, columnName)
		conditions = append(conditions, condition)
	}
	return strings.Join(conditions, " OR ")
}

func (c *ChangeHistoryManager) updateHistoryTable(originalTable, historyTable *tableMetadata) ([]string, error) {
	statements := make([]string, 0)

	// add new columns
	for _, column := range historyTable.columns {
		if column.exists {
			continue
		}

		statements = append(statements, fmt.Sprintf("ALTER TABLE %s ADD %s",
			historyTable.tableNameQuoted,
			column.definition,
		))
	}

	// remove untracked columns - without this, the triggers will not work
	historyColumnNames := historyTable.columnNames()
	dropColumns, err := c.buildDropColumns(originalTable.tableName, historyTable.tableName)
	if err != nil {
		return nil, fmt.Errorf("unable to fetch columns to be dropped: %w", err)
	}

	// keep columns to remove which are not part of the history table
	dropColumns, _ = funk.Difference(dropColumns, historyColumnNames)

	statements = append(statements, funk.Map(dropColumns, func(column string) string {
		return fmt.Sprintf("ALTER TABLE %s DROP COLUMN `%s`", historyTable.tableNameQuoted, column)
	})...)

	return statements, nil
}

func (c *ChangeHistoryManager) buildDropColumns(originalTable, historyTable string) ([]string, error) {
	qry := `
select history_entries.COLUMN_NAME
from (select COLUMN_NAME
      from information_schema.columns
      where table_schema = '%s' and TABLE_NAME = '%s') history_entries
         left join (select COLUMN_NAME
               from information_schema.columns
               where table_schema = '%s' and TABLE_NAME = '%s') as original
              on original.COLUMN_NAME = history_entries.COLUMN_NAME
where original.COLUMN_NAME IS NULL`

	results, err := c.getTableMetaData(1, func(database string) string {
		return fmt.Sprintf(qry, database, historyTable, database, originalTable)
	})
	if err != nil {
		return nil, err
	}

	return funk.Map(results, func(i []string) string {
		return i[0]
	}), nil
}

func (c *ChangeHistoryManager) getTableMetaData(columnLength int, queryBuilder func(database string) string) (results [][]string, err error) {
	dbName := c.orm.Dialect().CurrentDatabase()
	query := queryBuilder(dbName)

	db := c.orm.Raw(query)
	if db.Error != nil {
		return nil, fmt.Errorf("unable to query metadata: %w", db.Error)
	}

	var rows *sql.Rows
	rows, err = db.Rows()
	if err != nil {
		return nil, fmt.Errorf("cannot fetch rows: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			err = multierror.Append(err, fmt.Errorf("closing rows failed: %w", closeErr))
		}
	}()

	results = make([][]string, 0)
	for rows.Next() {
		result := make([]*string, columnLength)
		dest := make([]any, columnLength)
		for i := range result {
			dest[i] = &result[i]
		}

		err = rows.Scan(dest...)
		if err != nil {
			return nil, fmt.Errorf("unable to scan result row: %w", err)
		}

		results = append(results, funk.Map(result, mdl.EmptyIfNil[string]))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("fetching rows failed: %w", err)
	}

	return results, nil
}

func (c *ChangeHistoryManager) execute(statements []string) error {
	if !c.settings.MigrationEnabled {
		for _, statement := range statements {
			c.logger.Info("planned schema change: " + statement)
		}

		c.logger.Info("change history migration is disabled, please apply the changes manually")

		return fmt.Errorf("missing schema migrations (disabled)")
	}

	for _, statement := range statements {
		c.logger.Debug(statement)
		_, err := c.orm.CommonDB().Exec(statement)
		if err != nil {
			c.logger.WithFields(log.Fields{
				"sql": statement,
			}).Error("could not migrate change history: %w", err)

			return fmt.Errorf("could not migrate change history: %w", err)
		}
	}

	c.logger.Info("change history setup is now up to date")

	return nil
}
