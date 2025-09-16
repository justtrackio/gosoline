package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

type TableData []map[string]any

// DatabaseData example data:
//
//	map[string][]map[string]any{
//	  "users": {
//	    {
//	      "username": "alice",
//	    },
//	  },
//	}
type DatabaseData map[string]TableData

//go:generate go run github.com/vektra/mockery/v2 --name DataExporter
type DataExporter interface {
	ExportAllTables(ctx context.Context, dbName string) (DatabaseData, error)
	ExportTable(ctx context.Context, dbName string, tableName string) ([]map[string]any, error)
	Close() error
}

type exporterCtxKey string

func ProvideDataExporter(ctx context.Context, config cfg.Config, logger log.Logger, clientName string) (DataExporter, error) {
	return appctx.Provide(ctx, exporterCtxKey(clientName), func() (DataExporter, error) {
		return NewDataExporter(ctx, config, logger, clientName)
	})
}

func NewDataExporter(ctx context.Context, config cfg.Config, logger log.Logger, clientName string) (DataExporter, error) {
	client, err := NewClient(ctx, config, logger, clientName)
	if err != nil {
		return nil, fmt.Errorf("failed to provide db client: %w", err)
	}

	logger = logger.WithFields(log.Fields{
		"client": clientName,
	})

	return NewDataExporterWithInterfaces(logger, client), nil
}

func NewDataExporterWithInterfaces(logger log.Logger, client Client) DataExporter {
	return &dataExporter{
		logger: logger.WithChannel("db-data-exporter"),
		client: client,
		ignoredModels: []string{
			"fixture",
		},
	}
}

type dataExporter struct {
	logger        log.Logger
	client        Client
	ignoredModels []string
}

func (d *dataExporter) ExportTable(ctx context.Context, dbName string, tableName string) (data []map[string]any, err error) {
	log.AppendContextFields(ctx, map[string]any{
		"dbName":    dbName,
		"tableName": tableName,
	})
	d.logger.Info(ctx, "exporting model %s", tableName)
	defer d.logger.Info(ctx, "done exporting model %s", tableName)

	var resExists, resData *sqlx.Rows

	// check table is valid, as we can't place it as an argument in the select query (only condition values work here)
	if resExists, err = d.client.Queryx(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA=? AND TABLE_NAME=?", dbName, tableName); err != nil {
		return nil, fmt.Errorf("failed to check if table exists for model %s: %w", tableName, err)
	}
	defer func() {
		if closeErr := resExists.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close rows result: %w", closeErr))
		}
	}()

	if !resExists.Next() {
		orgErr := fmt.Errorf("model %s not found", tableName)

		return nil, orgErr
	}

	if resData, err = d.client.Queryx(ctx, fmt.Sprintf("SELECT * FROM `%s`.`%s`", dbName, tableName)); err != nil {
		return nil, fmt.Errorf("failed to select all data for model %s: %w", tableName, err)
	}
	defer func() {
		if closeErr := resData.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close rows result: %w", closeErr))
		}
	}()

	for resData.Next() {
		row := map[string]any{}

		if err = resData.MapScan(row); err != nil {
			return nil, fmt.Errorf("failed scan all data for model %s: %w", tableName, err)
		}

		for k, v := range row {
			// for now, we're exping strings in this case
			// this has to be improved for more complex data types
			if n, ok := v.([]uint8); ok {
				row[k] = string(n)
			}
		}

		data = append(data, row)
	}

	return data, nil
}

func (d *dataExporter) ExportAllTables(ctx context.Context, dbName string) (data DatabaseData, err error) {
	var res *sqlx.Rows

	if res, err = d.client.Queryx(ctx, "SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA=?", dbName); err != nil {
		return nil, fmt.Errorf("failed to check if table exists in database %s: %w", dbName, err)
	}
	defer func() {
		if closeErr := res.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("failed to close rows result: %w", closeErr))
		}
	}()

	models := make([]string, 0)
	// Let's first read all results, to hand back the connection to the pool as fast as possible
	for res.Next() {
		var model string

		err = res.Scan(&model)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row for table name in database %s: %w", dbName, err)
		}

		if funk.Contains(d.ignoredModels, model) {
			continue
		}

		models = append(models, model)
	}

	data = DatabaseData{}

	for _, model := range models {
		modelData, err := d.ExportTable(ctx, dbName, model)
		if err != nil {
			return nil, fmt.Errorf("failed to export model %s in database %s: %w", model, dbName, err)
		}

		data[model] = modelData
	}

	return data, nil
}

func (d *dataExporter) Close() error {
	if err := d.client.Close(); err != nil {
		return fmt.Errorf("failed to close database client: %w", err)
	}

	return nil
}
