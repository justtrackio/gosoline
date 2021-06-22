package fixtures

import (
	"context"
	"github.com/applike/gosoline/pkg/blob"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/aws/aws-sdk-go/aws"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type BlobFixturesSettings struct {
	BasePath   string
	ConfigName string
}

type blobFixtureWriter struct {
	logger      log.Logger
	batchRunner blob.BatchRunner
	purger      *blobPurger
	store       blob.Store
	basePath    string
}

func BlobFixtureWriterFactory(settings *BlobFixturesSettings) FixtureWriterFactory {
	return func(config cfg.Config, logger log.Logger) (FixtureWriter, error) {
		basePath, err := filepath.Abs(settings.BasePath)
		if err != nil {
			return nil, err
		}

		settings.BasePath = basePath

		store := blob.NewStore(config, logger, settings.ConfigName)
		br := blob.NewBatchRunner(config, logger)
		purger := newBlobPurger(config, logger, settings)

		return NewBlobFixtureWriterWithInterfaces(logger, br, purger, store, settings.BasePath), nil
	}
}

func NewBlobFixtureWriterWithInterfaces(logger log.Logger, batchRunner blob.BatchRunner, purger *blobPurger, store blob.Store, basePath string) FixtureWriter {
	return &blobFixtureWriter{
		logger:      logger,
		batchRunner: batchRunner,
		purger:      purger,
		store:       store,
		basePath:    basePath,
	}
}

func (s *blobFixtureWriter) Purge() error {
	return s.purger.purge()
}

func (s *blobFixtureWriter) Write(_ *FixtureSet) error {
	s.store.CreateBucket()

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

	var batch blob.Batch
	for _, file := range files {
		body, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		object := blob.Object{
			Key:  aws.String(strings.Replace(file, s.basePath, "", 1)),
			Body: blob.StreamBytes(body),
		}

		batch = append(batch, &object)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		err = s.batchRunner.Run(ctx)
	}(ctx)

	s.store.Write(batch)
	cancel()

	s.logger.Info("loaded %d files", len(files))

	return err
}
