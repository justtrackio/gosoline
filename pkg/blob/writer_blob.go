package blob

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/fixtures"
	"github.com/justtrackio/gosoline/pkg/log"
)

// BlobFixture is a dummy struct for writing nicely typed fixture sets, even though the blob fixture loader works a bit differently
type BlobFixture struct{}

//go:generate go run github.com/vektra/mockery/v2 --name Reader

// Reader provides a channel of Object that can be iterated over for writing fixtures
type Reader interface {
	Chan(ctx context.Context) (<-chan Object, error)
}

// FileReader reads files from a directory path, similar to the original basePath behavior
type FileReader struct {
	basePath string
}

// NewFileReader creates a new FileReader for the given base path
func NewFileReader(basePath string) (Reader, error) {
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}

	return &FileReader{basePath: absPath}, nil
}

// processFile processes a single file and sends it to the channel if successful
func (f *FileReader) processFile(ctx context.Context, ch chan Object, info os.FileInfo, path string) error {
	if info.IsDir() {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	key := f.generateKey(path)

	select {
	case ch <- Object{Key: &key, Body: StreamBytes(body)}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// generateKey creates a key from the file path by removing the base path and leading slash
func (f *FileReader) generateKey(path string) string {
	key := strings.Replace(path, f.basePath, "", 1)
	// Remove leading slash if present
	if key != "" && key[0] == '/' {
		key = key[1:]
	}

	return key
}

// Chan iterates through files in the base path and sends them over a channel
func (f *FileReader) Chan(ctx context.Context) (<-chan Object, error) {
	ch := make(chan Object)

	go func() {
		defer close(ch)

		err := filepath.Walk(f.basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			return f.processFile(ctx, ch, info, path)
		})
		if err != nil {
			// In case of error, we should log it, but we can't return it from a goroutine
			// The current implementation doesn't handle walk errors gracefully either
			// For now, we'll just stop processing
			return
		}
	}()

	return ch, nil
}

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

type ReaderFactory func() (Reader, error)

// FileReaderFactory creates a factory function for generating a file reader
func FileReaderFactory(basePath string) ReaderFactory {
	return func() (Reader, error) {
		reader, err := NewFileReader(basePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create file reader: %w", err)
		}

		return reader, nil
	}
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
		batch = append(batch, &object)
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

	s.logger.Info("loaded %d files", fileCount)

	return err
}
