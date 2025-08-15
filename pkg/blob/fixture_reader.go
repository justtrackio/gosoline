package blob

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:generate go run github.com/vektra/mockery/v2 --name Reader

// Reader provides a channel of Object that can be iterated over for writing fixtures
type Reader interface {
	Chan(ctx context.Context) (<-chan *Object, error)
	Source() string
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
func (f *FileReader) processFile(ctx context.Context, ch chan *Object, info os.FileInfo, path string) error {
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
	case ch <- &Object{Key: &key, Body: StreamBytes(body)}:
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
func (f *FileReader) Chan(ctx context.Context) (<-chan *Object, error) {
	ch := make(chan *Object)

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

func (f *FileReader) Source() string {
	return f.basePath
}
