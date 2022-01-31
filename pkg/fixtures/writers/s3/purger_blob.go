package s3

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type blobPurger struct {
	logger      log.Logger
	batchRunner blob.BatchRunner
	store       blob.Store
}

func newBlobPurger(ctx context.Context, config cfg.Config, logger log.Logger, settings *BlobFixturesSettings) (*blobPurger, error) {
	store, err := blob.NewStore(ctx, config, logger, settings.ConfigName)
	if err != nil {
		return nil, fmt.Errorf("can not create blob store: %w", err)
	}

	br, err := blob.NewBatchRunner(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create blob batch runner: %w", err)
	}

	return &blobPurger{
		logger:      logger,
		batchRunner: br,
		store:       store,
	}, nil
}

func (p *blobPurger) Purge(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)

	var batchRunnerErr error
	go func(ctx context.Context) {
		batchRunnerErr = p.batchRunner.Run(ctx)
	}(ctx)

	err := p.store.DeleteBucket(ctx)
	cancel()

	if batchRunnerErr != nil {
		return batchRunnerErr
	}

	return err
}
