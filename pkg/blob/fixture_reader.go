package blob

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
)

var _ FixtureReader = &fileReader{}

//go:generate go run github.com/vektra/mockery/v2 --name FixtureReader

// FixtureReader provides a channel of Object that can be iterated over for writing fixtures
type FixtureReader interface {
	Chan() <-chan *Object
	Run(ctx context.Context) error
	Stop()
	Source() string
}

type ReaderFactory func(ctx context.Context, config cfg.Config, logger log.Logger, storeName string) (FixtureReader, error)

// fileReader reads files from a directory path, similar to the original basePath behavior
type fileReader struct {
	ch        chan *Object
	closeOnce sync.Once
	basePath  string
}

// NewFileReader creates a new fileReader for the given base path
func NewFileReader(basePath string) ReaderFactory {
	return func(_ context.Context, _ cfg.Config, _ log.Logger, _ string) (FixtureReader, error) {
		absPath, err := filepath.Abs(basePath)
		if err != nil {
			return nil, err
		}

		return &fileReader{
			ch:       make(chan *Object),
			basePath: absPath,
		}, nil
	}
}

// processPath processes a single path (directory or file) and sends files to the channel if successful
func (f *fileReader) processPath(info os.FileInfo, path string) error {
	if info.IsDir() {
		return nil
	}

	body, err := os.Open(path)
	if err != nil {
		return err
	}

	key := f.generateKey(path)

	f.ch <- &Object{
		Key:  &key,
		Body: StreamReader(body),
	}

	return nil
}

// generateKey creates a key from the file path by removing the base path and leading slash
func (f *fileReader) generateKey(path string) string {
	key := strings.Replace(path, f.basePath, "", 1)
	// Remove leading slash if present
	if key != "" && key[0] == '/' {
		key = key[1:]
	}

	return key
}

// Chan iterates through files in the base path and sends them over a channel
func (f *fileReader) Chan() <-chan *Object {
	return f.ch
}

func (f *fileReader) Run(ctx context.Context) error {
	defer f.Stop()

	err := filepath.Walk(f.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if ok, err := exec.IsContextDone(ctx); ok {
			return err
		}

		return f.processPath(info, path)
	})

	return err
}

func (f *fileReader) Source() string {
	return f.basePath
}

func (f *fileReader) Stop() {
	f.closeOnce.Do(func() {
		close(f.ch)
	})
}
