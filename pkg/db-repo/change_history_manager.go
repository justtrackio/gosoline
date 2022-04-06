package db_repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type changeHistoryManagerSettings struct {
	ChangeAuthorField string `cfg:"change_author_column"`
	TableSuffix       string `cfg:"table_suffix" default:"history"`
}

type ChangeHistoryManager struct {
	orm      *gorm.DB
	logger   log.Logger
	settings *changeHistoryManagerSettings
	models   []ModelBased
}

type changeHistoryManagerAppctxKey int

func ProvideChangeHistoryManager(ctx context.Context, config cfg.Config, logger log.Logger) (*ChangeHistoryManager, error) {
	return appctx.Provide(ctx, changeHistoryManagerAppctxKey(0), func() (*ChangeHistoryManager, error) {
		return NewChangeHistoryManager(config, logger)
	})
}

func NewChangeHistoryManager(config cfg.Config, logger log.Logger) (*ChangeHistoryManager, error) {
	orm, err := NewOrm(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	settings := &changeHistoryManagerSettings{}
	config.UnmarshalKey("change_history", settings)

	return &ChangeHistoryManager{
		logger:   logger.WithChannel("change_history_manager"),
		orm:      orm,
		settings: settings,
	}, nil
}

func (c *ChangeHistoryManager) addModels(models ...ModelBased) {
	c.models = append(c.models, models...)
}

func (c *ChangeHistoryManager) RunMigrations() error {
	for _, model := range c.models {
		if err := c.RunMigration(model); err != nil {
			return fmt.Errorf("can not run migration for model %T: %w", model, err)
		}
	}

	return nil
}

func (c *ChangeHistoryManager) RunMigration(model ModelBased) error {
	statements := make([]string, 0)
	originalTable := c.buildOriginalTableMetadata(model)
	historyTable := c.buildHistoryTableMetadata(model, originalTable)

	if !historyTable.exists {
		statements = append(statements, c.createHistoryTable(historyTable))
		statements = append(statements, c.dropHistoryTriggers(originalTable, historyTable)...)
		statements = append(statements, c.createHistoryTriggers(originalTable, historyTable)...)
		c.logger.Info("creating change history setup")
		return c.execute(statements)
	}

	updated, statement := c.updateHistoryTable(historyTable)
	if updated {
		statements = append(statements, statement)
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
	modelFields := c.orm.NewScope(model).GetModelStruct().StructFields
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
			c.insertHistoryEntry(originalTable, historyTable, "insert", true),
			c.primaryKeysMatchCondition(originalTable, NewRecord),
		),
		fmt.Sprintf(`CREATE TRIGGER %s_au AFTER UPDATE ON %s FOR EACH ROW %s WHERE %s AND (%s)`,
			originalTable.tableName,
			originalTable.tableNameQuoted,
			c.insertHistoryEntry(originalTable, historyTable, "update", true),
			c.primaryKeysMatchCondition(originalTable, NewRecord),
			c.rowUpdatedCondition(originalTable),
		),
		fmt.Sprintf(`CREATE TRIGGER %s_bd BEFORE DELETE ON %s FOR EACH ROW %s WHERE %s`,
			originalTable.tableName,
			originalTable.tableNameQuoted,
			c.insertHistoryEntry(originalTable, historyTable, "delete", false),
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

func (c *ChangeHistoryManager) insertHistoryEntry(originalTable *tableMetadata, historyTable *tableMetadata, action string, includeAuthorEmail bool) string {
	columnNames := originalTable.columnNamesQuoted()
	if !includeAuthorEmail {
		columnNames = originalTable.columnNamesQuotedExcludingValue(c.settings.ChangeAuthorField)
	}

	columns := strings.Join(columnNames, ",")
	values := "d." + strings.Join(columnNames, ", d.")

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
	return fmt.Sprintf(`
		BEGIN 
			SET NEW.change_history_revision = (SELECT IFNULL(MAX(d.change_history_revision), 0) + 1 FROM %s as d WHERE %s); 
		END`,
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
	columnNames := originalTable.columnNamesQuotedExcludingValue(c.settings.ChangeAuthorField, ColumnUpdatedAt)
	var conditions []string
	for _, columnName := range columnNames {
		condition := fmt.Sprintf("NOT (OLD.%s <=> NEW.%s)", columnName, columnName)
		conditions = append(conditions, condition)
	}
	return strings.Join(conditions, " OR ")
}

func (c *ChangeHistoryManager) updateHistoryTable(historyTable *tableMetadata) (bool, string) {
	for _, column := range historyTable.columns {
		if column.exists {
			continue
		}

		return true, fmt.Sprintf("ALTER TABLE %s ADD %s",
			historyTable.tableNameQuoted,
			column.definition,
		)
	}

	return false, ""
}

func (c *ChangeHistoryManager) execute(statements []string) error {
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
