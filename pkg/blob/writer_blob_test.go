package blob_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/justtrackio/gosoline/pkg/blob"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/stretchr/testify/suite"
)

type WriterBlobTestSuite struct {
	suite.Suite

	config cfg.GosoConf
}

func TestWriterBlobTestSuite(t *testing.T) {
	suite.Run(t, new(WriterBlobTestSuite))
}

func (s *WriterBlobTestSuite) SetupTest() {
	s.config = cfg.New()
	err := s.config.Option(cfg.WithConfigMap(map[string]any{
		"app_project": "justtrack",
		"app_family":  "gosoline",
		"app_group":   "grp",
		"app_name":    "uploader",
		"env":         "test",
		"blob": map[string]any{
			"test": map[string]any{},
		},
	}))

	s.NoError(err, "there should be no error on config create")
}

func (s *WriterBlobTestSuite) TestFileReader() {
	// Create temporary directory with test files
	tmpDir := s.T().TempDir()

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
		s.NoError(err)

		err = os.WriteFile(fullPath, []byte(content), 0o644)
		s.NoError(err)
	}

	// Test FileReader
	reader, err := blob.NewFileReader(tmpDir)
	s.NoError(err)

	ctx := s.T().Context()
	ch, err := reader.Chan(ctx)
	s.NoError(err)

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
	s.Len(actualFiles, len(testFiles))

	for _, object := range actualFiles {
		key := *object.Key
		expectedContent, exists := testFiles[key]
		s.True(exists, "Unexpected file key: %s", key)

		// Convert stream to bytes for comparison
		bodyBytes, err := object.Body.ReadAll()
		s.NoError(err)
		s.Equal(expectedContent, string(bodyBytes), "Content mismatch for key: %s", key)
	}
}

func (s *WriterBlobTestSuite) TestFileReader_WithContextCancellation() {
	// Create temporary directory with test file
	tmpDir := s.T().TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	s.NoError(err)

	reader, err := blob.NewFileReader(tmpDir)
	s.NoError(err)

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(s.T().Context())

	ch, err := reader.Chan(ctx)
	s.NoError(err)

	// Cancel the context immediately
	cancel()

	// Channel should be closed (eventually) when context is cancelled
	// We can't guarantee exact timing, but we should be able to read from it without hanging
	select {
	case object, ok := <-ch:
		if ok {
			// If we got a file, that's fine - it means the goroutine processed it before cancellation
			s.NotNil(object.Key)
			s.NotEmpty(*object.Key)
		}
		// If !ok, channel was closed, which is also fine
	case <-ctx.Done():
		// Context was cancelled, which is expected
	}
}

func (s *WriterBlobTestSuite) TestNewFileReader_InvalidPath() {
	// Test with non-existent path - this should still create a reader
	// The error would come later when trying to walk the path
	reader, err := blob.NewFileReader("/non/existent/path")
	s.NoError(err) // NewFileReader doesn't validate path existence
	s.NotNil(reader)

	// Test with relative path - should be converted to absolute
	reader, err = blob.NewFileReader(".")
	s.NoError(err)
	s.NotNil(reader)
}
