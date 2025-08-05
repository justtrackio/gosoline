package blob

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

// BlobFixture is a dummy struct for writing nicely typed fixture sets, even though the blob fixture loader works a bit differently
type BlobFixture struct{}

type BlobFixturesSettings struct {
	BasePath   string
	ConfigName string
}

type blobFixtureWriter struct {
	logger      log.Logger
	batchRunner BatchRunner
	store       Store
	basePath    string
}

func BlobFixtureSetFactory[T any](settings *BlobFixturesSettings, data fixtures.NamedFixtures[T], options ...fixtures.FixtureSetOption) fixtures.FixtureSetFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (fixtures.FixtureSet, error) {
		var err error
		var writer fixtures.FixtureWriter

		if writer, err = NewBlobFixtureWriter(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("failed to create blob fixture writer: %w", err)
		}

		return fixtures.NewSimpleFixtureSet(data, writer, options...), nil
	}
}

func NewBlobFixtureWriter(ctx context.Context, config cfg.Config, logger log.Logger, settings *BlobFixturesSettings) (fixtures.FixtureWriter, error) {
	basePath, err := filepath.Abs(settings.BasePath)
	if err != nil {
		return nil, err
	}

	settings.BasePath = basePath

	store, err := NewStore(ctx, config, logger, settings.ConfigName)
	if err != nil {
		return nil, fmt.Errorf("can not create blob store: %w", err)
	}

	br, err := NewBatchRunner(ctx, config, logger, settings.ConfigName)
	if err != nil {
		return nil, fmt.Errorf("can not create blob batch runner: %w", err)
	}

	return NewBlobFixtureWriterWithInterfaces(logger, br, store, settings.BasePath), nil
}

func NewBlobFixtureWriterWithInterfaces(logger log.Logger, batchRunner BatchRunner, store Store, basePath string) fixtures.FixtureWriter {
	return &blobFixtureWriter{
		logger:      logger,
		batchRunner: batchRunner,
		store:       store,
		basePath:    basePath,
	}
}

func (s *blobFixtureWriter) Write(ctx context.Context, _ []any) error {
	var files []string
	err := filepath.Walk(s.basePath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	var batch Batch
	for _, file := range files {
		body, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		object := Object{
			Key:  aws.String(strings.Replace(file, s.basePath, "", 1)),
			Body: StreamBytes(body),
		}

		batch = append(batch, &object)
	}

	ctx, cancel := context.WithCancel(ctx)
	go coffin.RunLabeled(ctx, "fixtures/writerBlob", func() {
		err = s.batchRunner.Run(ctx)
	})
	defer cancel()

	if err := s.store.Write(batch); err != nil {
		return fmt.Errorf("can not write fixtes: %w", err)
	}

	s.logger.Info("loaded %d files", len(files))

	return err
}
