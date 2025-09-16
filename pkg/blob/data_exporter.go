package blob

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

//go:generate go run github.com/vektra/mockery/v2 --name DataExporter
type (
	DataExporter interface {
		ExportAllObjects(ctx context.Context) (StoreEntries, error)
	}
	exporterCtxKey string
	StoreEntries   []StoreEntry
	StoreEntry     struct {
		Key  string `json:"key"`
		Body []byte `json:"body"`
	}
	StoreProviderFn func(storeName string) (Store, error)
)

func ProvideDataExporter(ctx context.Context, config cfg.Config, logger log.Logger, storeName string) (DataExporter, error) {
	return appctx.Provide(ctx, exporterCtxKey(storeName), func() (DataExporter, error) {
		return newDataExporter(ctx, config, logger, storeName)
	})
}

func newDataExporter(ctx context.Context, config cfg.Config, logger log.Logger, storeName string) (DataExporter, error) {
	store, err := ProvideStore(ctx, config, logger, storeName)
	if err != nil {
		return nil, fmt.Errorf("unable to provide store %s: %w", storeName, err)
	}

	return &dataExporter{
		logger: logger.WithChannel("blob-data-exporter").
			WithFields(log.Fields{
				"store": storeName,
			}),
		store: store,
	}, nil
}

type dataExporter struct {
	logger log.Logger
	store  Store
}

func (d *dataExporter) ExportAllObjects(ctx context.Context) (data StoreEntries, err error) {
	d.logger.Info(ctx, "exporting all objects")
	defer d.logger.Info(ctx, "done exporting all objects")

	var objectBatch Batch
	if objectBatch, err = d.store.ListObjects(ctx, ""); err != nil {
		return nil, fmt.Errorf("could not list objects: %w", err)
	}

	d.store.Read(objectBatch)

	for _, object := range objectBatch {
		var b []byte
		if b, err = object.Body.ReadAll(); err != nil {
			return nil, fmt.Errorf("could not read object: %w", err)
		}

		data = append(data, StoreEntry{
			Key:  mdl.EmptyIfNil(object.Key),
			Body: b,
		})
	}

	return
}
