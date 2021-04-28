package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

type blobPurger struct {
	logger      mon.Logger
	batchRunner blob.BatchRunner
	store       blob.Store
}

func newBlobPurger(config cfg.Config, logger mon.Logger, settings *BlobFixturesSettings) *blobPurger {
	store := blob.NewStore(config, logger, settings.ConfigName)
	br := blob.NewBatchRunner(config, logger)

	return &blobPurger{
		logger:      logger,
		batchRunner: br,
		store:       store,
	}
}

func (p *blobPurger) purge() error {
	ctx, cancel := context.WithCancel(context.Background())

	var batchRunnerErr error
	go func(ctx context.Context) {
		batchRunnerErr = p.batchRunner.Run(ctx)
	}(ctx)

	err := p.store.DeleteBucket()
	cancel()

	if batchRunnerErr != nil {
		return batchRunnerErr
	}

	return err
}
