package db_repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"

	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ClientGorm struct {
	client db.Client
}

func NewClientGorm(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*ClientGorm, error) {
	client, err := db.NewClient(ctx, config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	return NewClientGormWithInterfaces(client), nil
}

//func NewClientGormWithSettings(ctx context.Context, config cfg.Config, logger log.Logger, name string, settings *Settings, options ...ClientOption) (*ClientSqlx, error) {
//	var (
//		err        error
//		connection *sqlx.DB
//		executor   exec.Executor = exec.NewDefaultExecutor()
//	)
//
//	if connection, err = ProvideConnectionFromSettings(ctx, logger, name, settings); err != nil {
//		return nil, fmt.Errorf("can not connect to sql database: %w", err)
//	}
//
//	if settings.Retry.Enabled {
//		executor = NewExecutor(config, logger, name, ExecutorBackoffType(name))
//	}
//
//	client := NewClientWithInterfaces(logger, connection, executor)
//
//	for _, option := range options {
//		option(client)
//	}
//
//	return client, nil
//}

func NewClientGormWithInterfaces(client db.Client) *ClientGorm {
	return &ClientGorm{
		client: client,
	}
}

func (c *ClientGorm) Exec(query string, args ...any) (sql.Result, error) {
	return c.client.Exec(context.Background(), query, args...)
}

func (c *ClientGorm) Prepare(query string) (*sql.Stmt, error) {
	return c.client.Prepare(context.Background(), query)
}

func (c *ClientGorm) Query(query string, args ...any) (*sql.Rows, error) {
	return c.client.Query(context.Background(), query, args...)
}

func (c *ClientGorm) QueryRow(query string, args ...any) *sql.Row {
	return c.client.QueryRow(context.Background(), query, args...)
}
