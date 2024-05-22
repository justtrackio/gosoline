package mocks_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/stretchr/testify/assert"
)

func TestLoggerMock_RecordsCorrectLogs(t *testing.T) {
	logger := mocks.NewLoggerMockedAll().WithBufferSize(10)

	for i := 0; i < 50; i++ {
		logger.Info("%d", i)

		buffered := logger.BufferedLogs()
		assert.Len(t, buffered, min(i+1, 10))
		for recordI, record := range buffered {
			assert.Equal(t, i-len(buffered)+1+recordI, record.Arguments[0])
		}
	}
}
