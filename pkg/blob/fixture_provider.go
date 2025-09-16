package blob

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/fixtures/provider"
)

const FixtureProviderName = "blob"

func blobName(m Metadata) string {
	return m.Name
}

func blobExport(ctx context.Context, exporter DataExporter, metadata Metadata) (StoreEntries, error) {
	data, err := exporter.ExportAllObjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export all objects from bucket %s: %w", metadata.Bucket, err)
	}

	return data, nil
}

func init() {
	provider.AddFixtureProviderFactory(FixtureProviderName, provider.NewFixtureProviderFactory(
		MetadataKey,
		ProvideDataExporter,
		blobName,
		blobExport,
	))
}
