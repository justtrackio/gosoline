package fixtures

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/log"
)

const (
	foreignKeyChecksStatement = "SET FOREIGN_KEY_CHECKS=%d;"
	truncateTableStatement    = "TRUNCATE TABLE %s;"
)

type mysqlPurger struct {
	client    db.Client
	logger    log.Logger
	tableName string
}

func newMysqlPurger(config cfg.Config, logger log.Logger, tableName string) (*mysqlPurger, error) {
	client, err := db.NewClient(config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create db client: %w", err)
	}

	return &mysqlPurger{client: client, logger: logger, tableName: tableName}, nil
}

func (p *mysqlPurger) purgeMysql() error {
	err := p.setForeignKeyChecks(0)

	if err != nil {
		p.logger.Error("error disabling foreign key checks: %w", err)

		return err
	}

	defer func() {
		err := p.setForeignKeyChecks(1)

		if err != nil {
			p.logger.Error("error enabling foreign key checks: %w", err)
		}
	}()

	_, err = p.client.Exec(fmt.Sprintf(truncateTableStatement, p.tableName))

	if err != nil {
		p.logger.Error("error truncating table %s: %w", p.tableName, err)
		return err
	}

	return nil
}

func (p *mysqlPurger) setForeignKeyChecks(enabled int) error {
	_, err := p.client.Exec(fmt.Sprintf(foreignKeyChecksStatement, enabled))

	return err
}
