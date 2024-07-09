package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
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

func newMysqlPurger(ctx context.Context, config cfg.Config, logger log.Logger, tableName string) (*mysqlPurger, error) {
	client, err := db.ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create db client: %w", err)
	}

	return &mysqlPurger{client: client, logger: logger, tableName: tableName}, nil
}

func (p *mysqlPurger) purgeMysql(ctx context.Context) error {
	//err := p.setForeignKeyChecks(0)
	//if err != nil {
	//	p.logger.Error("error disabling foreign key checks: %w", err)
	//
	//	return err
	//}
	//
	//defer func() {
	//	err := p.setForeignKeyChecks(1)
	//	if err != nil {
	//		p.logger.Error("error enabling foreign key checks: %w", err)
	//	}
	//}()
	//
	//_, err = p.client.Exec(ctx, fmt.Sprintf(truncateTableStatement, p.tableName))
	//if err != nil {
	//	p.logger.Error("error truncating table %s: %w", p.tableName, err)
	//	return err
	//}

	_, err := p.client.ExecMultiInTx(ctx, []db.Sqler{
		db.SqlFmt(foreignKeyChecksStatement, 0),
		db.SqlFmt(truncateTableStatement, p.tableName),
		db.SqlFmt(foreignKeyChecksStatement, 1),
	}...)
	if err != nil {
		p.logger.Error("error truncating table %s: %w", p.tableName, err)
		return err
	}

	return nil
}

func (p *mysqlPurger) setForeignKeyChecks(enabled int) error {
	ctx := context.Background()
	_, err := p.client.Exec(ctx, fmt.Sprintf(foreignKeyChecksStatement, enabled))

	return err
}
