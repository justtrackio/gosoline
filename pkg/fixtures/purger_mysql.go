package fixtures

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/mon"
)

const (
	foreignKeyChecksStatement = "SET FOREIGN_KEY_CHECKS=%d;"
	truncateTableStatement    = "TRUNCATE TABLE %s;"
)

type mysqlPurger struct {
	client    db.Client
	logger    mon.Logger
	tableName string
}

func newMysqlPurger(config cfg.Config, logger mon.Logger, tableName string) *mysqlPurger {
	client := db.NewClient(config, logger)

	return &mysqlPurger{client: client, logger: logger, tableName: tableName}
}

func (p *mysqlPurger) purgeMysql() error {
	err := p.setForeignKeyChecks(0)

	if err != nil {
		p.logger.Error(err, "error disabling foreign key checks")

		return err
	}

	defer func() {
		err := p.setForeignKeyChecks(1)

		if err != nil {
			p.logger.Error(err, "error enabling foreign key checks")
		}
	}()

	_, err = p.client.Exec(fmt.Sprintf(truncateTableStatement, p.tableName))

	if err != nil {
		p.logger.Errorf(err, "error truncating table %s", p.tableName)
		return err
	}

	return nil
}

func (p *mysqlPurger) setForeignKeyChecks(enabled int) error {
	_, err := p.client.Exec(fmt.Sprintf(foreignKeyChecksStatement, enabled))

	return err
}
