package db_repo

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/jinzhu/gorm"
	"reflect"
	"strings"
)

type changeHistoryManagerSettings struct {
	ChangeAuthorField string `cfg:"change_author_column"`
	TableSuffix       string `cfg:"table_suffix" default:"history"`
}

type changeHistoryManager struct {
	orm           *gorm.DB
	logger        log.Logger
	settings      *changeHistoryManagerSettings
	model         ModelBased
	originalTable *tableMetadata
	historyTable  *tableMetadata
	statements    []string
}

func newChangeHistoryManager(config cfg.Config, logger log.Logger, model ModelBased) (*changeHistoryManager, error) {
	orm, err := NewOrm(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create orm: %w", err)
	}

	settings := &changeHistoryManagerSettings{}
	config.UnmarshalKey("change_history", settings)

	return newChangeHistoryManagerWithInterfaces(logger, orm, model, settings), nil
}

func newChangeHistoryManagerWithInterfaces(logger log.Logger, orm *gorm.DB, model ModelBased, settings *changeHistoryManagerSettings) *changeHistoryManager {
	statements := make([]string, 0)

	logger = logger.WithChannel("change_history_manager").WithFields(log.Fields{
		"model": reflect.TypeOf(model).Elem().Name(),
	})

	return &changeHistoryManager{
		logger:     logger,
		orm:        orm,
		model:      model,
		statements: statements,
		settings:   settings,
	}
}

func (c *changeHistoryManager) runMigration() error {
	c.buildOriginalTableMetadata()
	c.buildHistoryTableMetadata()

	if !c.historyTable.exists {
		c.createHistoryTable()
		c.dropHistoryTriggers()
		c.createHistoryTriggers()
		c.logger.Info("creating change history setup")
		return c.execute()
	}

	updated := c.updateHistoryTable()
	if updated {
		c.dropHistoryTriggers()
		c.createHistoryTriggers()
		c.logger.Info("updating change history setup")
		return c.execute()
	}

	c.logger.Info("change history setup was already up to date")

	return nil
}

func (c *changeHistoryManager) buildOriginalTableMetadata() {
	scope := c.orm.NewScope(c.model)
	fields := scope.GetModelStruct().StructFields
	tableName := scope.TableName()

	c.originalTable = newTableMetadata(scope, tableName, fields)
}

func (c *changeHistoryManager) buildHistoryTableMetadata() {
	historyScope := c.orm.NewScope(ChangeHistoryModel{})
	tableName := fmt.Sprintf("%s_%s", c.originalTable.tableName, c.settings.TableSuffix)
	modelFields := c.orm.NewScope(c.model).GetModelStruct().StructFields
	fields := append(historyScope.GetModelStruct().StructFields, modelFields...)

	c.historyTable = newTableMetadata(historyScope, tableName, fields)
}

func (c *changeHistoryManager) createHistoryTable() {
	c.appendStatement(fmt.Sprintf("CREATE TABLE %v (%v, PRIMARY KEY (%v))",
		c.historyTable.tableNameQuoted,
		strings.Join(c.historyTable.columnDefinitions(), ","),
		strings.Join(c.historyTable.primaryKeyNamesQuoted(), ","),
	))
}

func (c *changeHistoryManager) appendStatement(statement string) {
	c.statements = append(c.statements, statement)
}

func (c *changeHistoryManager) dropHistoryTriggers() {
	triggers := []string{
		c.originalTable.tableName + "_ai",
		c.originalTable.tableName + "_au",
		c.originalTable.tableName + "_bd",
		c.historyTable.tableName + "_revai",
	}

	for _, trigger := range triggers {
		c.appendStatement(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s`, trigger))
	}
}

func (c *changeHistoryManager) createHistoryTriggers() {
	const NewRecord = "NEW"
	const OldRecord = "OLD"

	c.appendStatement(fmt.Sprintf(`CREATE TRIGGER %s_ai AFTER INSERT ON %s FOR EACH ROW %s WHERE %s`,
		c.originalTable.tableName,
		c.originalTable.tableNameQuoted,
		c.insertHistoryEntry("insert", true),
		c.primaryKeysMatchCondition(NewRecord),
	))

	c.appendStatement(fmt.Sprintf(`CREATE TRIGGER %s_au AFTER UPDATE ON %s FOR EACH ROW %s WHERE %s AND (%s)`,
		c.originalTable.tableName,
		c.originalTable.tableNameQuoted,
		c.insertHistoryEntry("update", true),
		c.primaryKeysMatchCondition(NewRecord),
		c.rowUpdatedCondition(),
	))

	c.appendStatement(fmt.Sprintf(`CREATE TRIGGER %s_bd BEFORE DELETE ON %s FOR EACH ROW %s WHERE %s`,
		c.originalTable.tableName,
		c.originalTable.tableNameQuoted,
		c.insertHistoryEntry("delete", false),
		c.primaryKeysMatchCondition(OldRecord),
	))

	c.appendStatement(fmt.Sprintf(`CREATE TRIGGER %s_revai BEFORE INSERT ON %s FOR EACH ROW %s`,
		c.historyTable.tableName,
		c.historyTable.tableNameQuoted,
		c.incrementRevision(),
	))
}

func (c *changeHistoryManager) insertHistoryEntry(action string, includeAuthorEmail bool) string {
	columnNames := c.originalTable.columnNamesQuoted()
	if !includeAuthorEmail {
		columnNames = c.originalTable.columnNamesQuotedExcludingValue(c.settings.ChangeAuthorField)
	}

	columns := strings.Join(columnNames, ",")
	values := "d." + strings.Join(columnNames, ", d.")

	return fmt.Sprintf(`
		INSERT INTO %s (change_history_action,change_history_revision,change_history_action_at,%s) 
			SELECT '%s', NULL, NOW(), %s 
			FROM %s AS d`,
		c.historyTable.tableNameQuoted,
		columns,
		action,
		values,
		c.originalTable.tableNameQuoted)
}

func (c *changeHistoryManager) incrementRevision() string {
	return fmt.Sprintf(`
		BEGIN 
			SET NEW.change_history_revision = (SELECT IFNULL(MAX(d.change_history_revision), 0) + 1 FROM %s as d WHERE %s); 
		END`,
		c.historyTable.tableNameQuoted,
		c.primaryKeysMatchCondition("NEW"),
	)
}

func (c *changeHistoryManager) primaryKeysMatchCondition(record string) string {
	var conditions []string
	for _, columnName := range c.originalTable.primaryKeyNamesQuoted() {
		condition := fmt.Sprintf("d.%s = %s.%s", columnName, record, columnName)
		conditions = append(conditions, condition)
	}
	return strings.Join(conditions, " AND ")
}

func (c *changeHistoryManager) rowUpdatedCondition() string {
	columnNames := c.originalTable.columnNamesQuotedExcludingValue(c.settings.ChangeAuthorField, ColumnUpdatedAt)
	var conditions []string
	for _, columnName := range columnNames {
		condition := fmt.Sprintf("NOT (OLD.%s <=> NEW.%s)", columnName, columnName)
		conditions = append(conditions, condition)
	}
	return strings.Join(conditions, " OR ")
}

func (c *changeHistoryManager) updateHistoryTable() bool {
	for _, column := range c.historyTable.columns {
		if column.exists {
			continue
		}

		c.appendStatement(fmt.Sprintf("ALTER TABLE %s ADD %s",
			c.historyTable.tableNameQuoted,
			column.definition,
		))
		return true
	}

	return false
}

func (c *changeHistoryManager) execute() error {
	for _, statement := range c.statements {
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
