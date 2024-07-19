package fixtures

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	foreignKeyChecksStatement = "SET FOREIGN_KEY_CHECKS=?;"
	truncateTableStatement    = "TRUNCATE TABLE %s;"
)

type mysqlPurger struct {
	client    db.Client
	logger    log.Logger
	tableName string
}

func NewMysqlPurger(ctx context.Context, config cfg.Config, logger log.Logger, tableName string) (*mysqlPurger, error) {
	client, err := db.ProvideClient(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("can not create db client: %w", err)
	}

	return &mysqlPurger{
		client:    client,
		logger:    logger.WithChannel("mysql-purger"),
		tableName: tableName,
	}, nil
}

func (p *mysqlPurger) Purge(ctx context.Context) error {
	_, err := p.client.ExecMultiInTx(ctx, []db.Sqler{
		db.SqlFmt(foreignKeyChecksStatement, nil, 0),
		db.SqlFmt(truncateTableStatement, []any{p.tableName}),
		db.SqlFmt(foreignKeyChecksStatement, nil, 1),
	}...)
	if err != nil {
		p.logger.Error("error truncating table %s: %w", p.tableName, err)
		return err
	}

	return nil
}
