package blob

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

var (
	_ Reader                 = &FileReader{}
	_ fixtures.FixtureWriter = &blobFixtureWriter{}
)

// BlobFixture is a dummy struct for writing nicely typed fixture sets, even though the blob fixture loader works a bit differently
type BlobFixture struct{}

type BlobFixturesSettings struct {
	Reader     Reader
	BasePath   string // Deprecated: use Reader instead, e.g. `NewFileReader(settings.BasePath)`
	ConfigName string
}

type blobFixtureWriter struct {
	logger      log.Logger
	batchRunner BatchRunner
	store       Store
	reader      Reader
}

func BlobFixtureSetFactory[T any](readerFactory ReaderFactory, configName string, options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var data fixtures.NamedFixtures[T]
		var reader Reader
		var writer fixtures.FixtureWriter

		if reader, err = readerFactory(); err != nil {
			return nil, fmt.Errorf("failed to create reader")
		}

		settings := &BlobFixturesSettings{
			Reader:     reader,
			ConfigName: configName,
		}

		if writer, err = NewBlobFixtureWriter(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("failed to create blob fixture writer: %w", err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewBlobFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, settings *BlobFixturesSettings) (fixtures.FixtureWriter, error) {
	store, err := NewStore(ctx, config, logger, settings.ConfigName)
	if err != nil {
		return nil, fmt.Errorf("can not create blob store: %w", err)
	}

	bri, err := NewBatchRunner(ctx, config, logger, settings.ConfigName)
	if err != nil {
		return nil, fmt.Errorf("can not create blob batch runner: %w", err)
	}

	reader := settings.Reader
	if reader == nil {
		return nil, fmt.Errorf("Reader must be provided")
	}

	return NewBlobFixtureWriterWithInterfaces(logger, bri, store, reader), nil
}

func NewBlobFixtureWriterWithInterfaces(logger log.Logger, batchRunner BatchRunner, store Store, reader Reader) fixtures.FixtureWriter {
	return &blobFixtureWriter{
		logger:      logger,
		batchRunner: batchRunner,
		store:       store,
		reader:      reader,
	}
}

func (s *blobFixtureWriter) Write(ctx context.Context, _ []any) error {
	readCh, err := s.reader.Chan(ctx)
	if err != nil {
		return fmt.Errorf("failed to read blob fixtures: %w", err)
	}

	var batch Batch
	fileCount := 0

	for object := range readCh {
		batch = append(batch, object)
		fileCount++
	}

	if fileCount == 0 {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	go func(ctx context.Context) {
		err = s.batchRunner.Run(ctx)
	}(ctx)
	defer cancel()

	if err := s.store.Write(batch); err != nil {
		return fmt.Errorf("can not write fixtes: %w", err)
	}

	s.logger.Info("loaded %d files from %s", fileCount, s.reader.Source())

	return err
}
