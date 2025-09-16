package db

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/fixtures/provider"
)

const FixtureProviderName = "db"

func dbName(m Metadata) string {
	return m.Name
}

func dbExport(ctx context.Context, exporter DataExporter, metadata Metadata) (DatabaseData, error) {
	data, err := exporter.ExportAllTables(ctx, metadata.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to export all tables from db %s: %w", metadata.Database, err)
	}

	return data, nil
}

func init() {
	provider.AddFixtureProviderFactory(FixtureProviderName, provider.NewFixtureProviderFactory(
		MetadataKey,
		ProvideDataExporter,
		dbName,
		dbExport,
	))
}
