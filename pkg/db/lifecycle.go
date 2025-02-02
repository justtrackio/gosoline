package db

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/reslife"
)

var tableExcludes = []string{
	"fixture",
	"goose_db_version",
}

func NewLifecycleManager(settings *Settings) func() (string, reslife.LifeCycleerFactory) {
	return func() (string, reslife.LifeCycleerFactory) {
		id := fmt.Sprintf("db/%s", settings.Uri.Database)

		return id, func(ctx context.Context, config cfg.Config, logger log.Logger) (reslife.LifeCycleer, error) {
			var err error
			var db *sqlx.DB

			fkSettings := *settings
			fkSettings.Parameters = map[string]string{
				"FOREIGN_KEY_CHECKS": "0",
			}
			for k, v := range settings.Parameters {
				fkSettings.Parameters[k] = v
			}

			if db, err = NewConnectionWithInterfaces(logger, &fkSettings); err != nil {
				return nil, fmt.Errorf("could not connect to database: %w", err)
			}

			return &LifecycleManager{
				logger:   logger,
				db:       db,
				settings: settings,
			}, nil
		}
	}
}

type LifecycleManager struct {
	logger   log.Logger
	db       *sqlx.DB
	settings *Settings
}

func (m *LifecycleManager) Create(ctx context.Context) error {
	return nil
}

func (l *LifecycleManager) Register(ctx context.Context) (string, any, error) {
	return "", nil, nil
}

func (m *LifecycleManager) Purge(ctx context.Context) error {
	var err error
	var table string
	var tables []string

	rows, err := m.db.QueryContext(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?;", m.settings.Uri.Database)
	if err != nil {
		return fmt.Errorf("failed to check tables of database: %w", err)
	}

	for rows.Next() {
		if err = rows.Scan(&table); err != nil {
			return fmt.Errorf("could not scan row: %w", err)
		}
		tables = append(tables, table)
	}

	if err = rows.Close(); err != nil {
		return fmt.Errorf("could not close rows: %w", err)
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("could not iterate over rows: %w", err)
	}

	tables = funk.Filter(tables, func(s string) bool {
		return !funk.Contains(tableExcludes, s)
	})
	chunks := funk.Chunk(tables, int(math.Ceil(float64(len(tables))/float64(runtime.NumCPU()))))

	cfn := coffin.New()
	for _, chunk := range chunks {
		cfn.GoWithContext(ctx, func(ctx context.Context) error {
			var sqls []string

			for _, table = range chunk {
				sqls = append(sqls, fmt.Sprintf("TRUNCATE TABLE %s;", table))
			}

			if _, err = m.db.ExecContext(ctx, strings.Join(sqls, " ")); err != nil {
				return fmt.Errorf("could not truncate tables: %w", err)
			}

			return nil
		})
	}

	if err = cfn.Wait(); err != nil {
		return fmt.Errorf("error while truncating tables: %w", err)
	}

	if err = m.db.Close(); err != nil {
		return fmt.Errorf("could not close database: %w", err)
	}

	return nil
}
