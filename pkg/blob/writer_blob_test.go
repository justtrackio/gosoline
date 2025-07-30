package blob_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileReader(t *testing.T) {
	// Create temporary directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := map[string]string{
		"file1.txt":        "content1",
		"subdir/file2.txt": "content2",
		"file3.dat":        "content3",
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tmpDir, filePath)
		dir := filepath.Dir(fullPath)

		err := os.MkdirAll(dir, 0o755)
		require.NoError(t, err)

		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Test FileReader
	reader, err := blob.NewFileReader(tmpDir)
	require.NoError(t, err)

	ctx := t.Context()
	ch, err := reader.Chan(ctx)
	require.NoError(t, err)

	// Collect all files
	var actualFiles []blob.Object
	for object := range ch {
		actualFiles = append(actualFiles, object)
	}

	// Sort for consistent comparison
	sort.Slice(actualFiles, func(i, j int) bool {
		return *actualFiles[i].Key < *actualFiles[j].Key
	})

	// Verify we got all expected files
	assert.Len(t, actualFiles, len(testFiles))

	for _, object := range actualFiles {
		key := *object.Key
		expectedContent, exists := testFiles[key]
		assert.True(t, exists, "Unexpected file key: %s", key)
		
		// Convert stream to bytes for comparison
		bodyBytes, err := object.Body.ReadAll()
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(bodyBytes), "Content mismatch for key: %s", key)
	}
}

func TestFileReader_WithContextCancellation(t *testing.T) {
	// Create temporary directory with test file
	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	reader, err := blob.NewFileReader(tmpDir)
	require.NoError(t, err)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(t.Context())

	ch, err := reader.Chan(ctx)
	require.NoError(t, err)

	// Cancel the context immediately
	cancel()

	// Channel should be closed (eventually) when context is cancelled
	// We can't guarantee exact timing, but we should be able to read from it without hanging
	select {
	case object, ok := <-ch:
		if ok {
			// If we got a file, that's fine - it means the goroutine processed it before cancellation
			assert.NotNil(t, object.Key)
			assert.NotEmpty(t, *object.Key)
		}
		// If !ok, channel was closed, which is also fine
	case <-ctx.Done():
		// Context was cancelled, which is expected
	}
}

func TestNewFileReader_InvalidPath(t *testing.T) {
	// Test with non-existent path - this should still create a reader
	// The error would come later when trying to walk the path
	reader, err := blob.NewFileReader("/non/existent/path")
	assert.NoError(t, err) // NewFileReader doesn't validate path existence
	assert.NotNil(t, reader)

	// Test with relative path - should be converted to absolute
	reader, err = blob.NewFileReader(".")
	assert.NoError(t, err)
	assert.NotNil(t, reader)
}

func TestBlobFixturesSettings_BackwardCompatibility(t *testing.T) {
	// Test that both BasePath and Reader can be set
	settings := &blob.BlobFixturesSettings{
		BasePath:   "/some/path",
		ConfigName: "test",
		Reader:     nil,
	}

	assert.Equal(t, "/some/path", settings.BasePath)
	assert.Equal(t, "test", settings.ConfigName)
	assert.Nil(t, settings.Reader)

	// Test with Reader
	reader, err := blob.NewFileReader("/tmp")
	require.NoError(t, err)

	settings.Reader = reader
	assert.NotNil(t, settings.Reader)
}

func TestNewBlobFixtureWriter_ErrorHandling(t *testing.T) {
	ctx := t.Context()
	config := cfg.New()
	logger := log.NewLogger()

	// Test error when neither BasePath nor Reader is provided
	settings := &blob.BlobFixturesSettings{
		ConfigName: "test",
		// Neither BasePath nor Reader set
	}

	_, err := blob.NewBlobFixtureWriter(ctx, config, logger, settings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "either Reader or BasePath must be provided")
}
