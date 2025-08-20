package blob

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
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
	var files int

	var err error

	bgCtx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		err = s.batchRunner.Run(ctx)
	}(bgCtx)
	defer cancel()

	err = filepath.Walk(s.basePath, func(path string, f os.FileInfo, err error) error {
		if ok, err := exec.IsContextDone(ctx); ok {
			return err
		}

		if err != nil {
			return err
		}

		if f.IsDir() {
			return nil
		}

		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		object := Object{
			Key:  aws.String(strings.Replace(path, s.basePath, "", 1)),
			Body: StreamBytes(body),
		}

		files++

		return s.store.WriteOne(&object)
	})
	if err != nil {
		return err
	}

	s.logger.Info("loaded %d files", files)

	return nil
}
