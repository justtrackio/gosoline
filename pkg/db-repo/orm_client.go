package db_repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/log"
)

type OrmClient struct {
	client db.Client
}

func NewOrmClient(ctx context.Context, config cfg.Config, logger log.Logger, name string) (*OrmClient, error) {
	client, err := db.NewClient(ctx, config, logger, name)
	if err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	return NewOrmClientWithInterfaces(client), nil
}

func NewOrmClientWithSettings(ctx context.Context, config cfg.Config, logger log.Logger, name string, settings *db.Settings, options ...db.ClientOption) (*OrmClient, error) {
	client, err := db.NewClientWithSettings(ctx, config, logger, name, settings, options...)
	if err != nil {
		return nil, fmt.Errorf("can not create client: %w", err)
	}

	clientGorm := NewOrmClientWithInterfaces(client)

	return clientGorm, nil
}

func NewOrmClientWithInterfaces(client db.Client) *OrmClient {
	return &OrmClient{
		client: client,
	}
}

func (c *OrmClient) Exec(query string, args ...any) (sql.Result, error) {
	return c.client.Exec(context.Background(), query, args...)
}

func (c *OrmClient) Prepare(query string) (*sql.Stmt, error) {
	return c.client.Prepare(context.Background(), query)
}

func (c *OrmClient) Query(query string, args ...any) (*sql.Rows, error) {
	return c.client.Query(context.Background(), query, args...)
}

func (c *OrmClient) QueryRow(query string, args ...any) *sql.Row {
	return c.client.QueryRow(context.Background(), query, args...)
}
