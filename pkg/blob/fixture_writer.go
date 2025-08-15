package blob

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

var _ fixtures.FixtureWriter = &blobFixtureWriter{}

// BlobFixture is a dummy struct for writing nicely typed fixture sets, even though the blob fixture loader works a bit differently
type BlobFixture struct{}

type blobFixtureWriter struct {
	logger      log.Logger
	batchRunner BatchRunner
	reader      FixtureReader
	store       Store
}

func BlobFixtureSetFactory[T any](readerFactory ReaderFactory, storeName string, options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var data fixtures.NamedFixtures[T]
		var writer fixtures.FixtureWriter

		if writer, err = NewBlobFixtureWriter(ctx, config, logger, readerFactory, storeName); err != nil {
			return nil, fmt.Errorf("failed to create blob fixture writer: %w", err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewBlobFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, readerFactory ReaderFactory, storeName string) (fixtures.FixtureWriter, error) {
	if readerFactory == nil {
		return nil, fmt.Errorf("reader must be provided")
	}

	logger = logger.WithFields(log.Fields{
		"store-name": storeName,
	})

	var err error
	var reader FixtureReader

	bri, err := NewBatchRunner(ctx, config, logger, storeName)
	if err != nil {
		return nil, fmt.Errorf("can not create blob batch runner: %w", err)
	}

	if reader, err = readerFactory(ctx, config, logger, storeName); err != nil {
		return nil, fmt.Errorf("failed to create blob fixture reader: %w", err)
	}

	store, err := NewStore(ctx, config, logger, storeName)
	if err != nil {
		return nil, fmt.Errorf("can not create blob store: %w", err)
	}

	return NewBlobFixtureWriterWithInterfaces(logger, bri, reader, store), nil
}

func NewBlobFixtureWriterWithInterfaces(logger log.Logger, batchRunner BatchRunner, reader FixtureReader, store Store) fixtures.FixtureWriter {
	return &blobFixtureWriter{
		logger:      logger,
		batchRunner: batchRunner,
		reader:      reader,
		store:       store,
	}
}

func (s *blobFixtureWriter) Write(ctx context.Context, _ []any) error {
	var err error
	objectCount := 0
	logger := s.logger.WithContext(ctx)

	writerCtx, cancelWriter := context.WithCancel(context.Background())

	cfn := coffin.New()
	cfn.GoWithContext(writerCtx, func(ctx context.Context) error {
		defer cancelWriter()
		defer s.reader.Stop()

		return s.batchRunner.Run(ctx)
	})
	cfn.GoWithContext(writerCtx, func(ctx context.Context) error {
		defer cancelWriter()
		defer s.reader.Stop()

		for object := range s.reader.Chan() {
			var ok bool

			// read until channel is empty and intentionally leaving error behind until end of function picks it up
			if ok, err = exec.IsContextDone(ctx); err != nil || ok {
				continue
			}

			if err = s.store.WriteOne(object); err != nil {
				return fmt.Errorf("can not write fixture: %w", err)
			}

			objectCount++
		}

		return nil
	})
	cfn.GoWithContext(ctx, s.reader.Run)

	err = cfn.Wait()
	if err != nil {
		return fmt.Errorf("can not write fixtures: %w", err)
	}

	logger.Info("loaded %d objects from %s", objectCount, s.reader.Source())

	return nil
}
