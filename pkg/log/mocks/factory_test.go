package mocks_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/log/mocks"
)

func TestFactory(t *testing.T) {
	logger := mocks.NewLoggerMock(mocks.WithTestingT(t), mocks.WithMockAll)
	logger.Debug(t.Context(), "debug message")
	logger.Info(t.Context(), "info message")
	logger.Warn(t.Context(), "warn message")
	logger.Error(t.Context(), "error message")
}
