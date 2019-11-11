package compression_test

import (
	"github.com/applike/gosoline/pkg/compression"
	"github.com/stretchr/testify/assert"
	"testing"
)

const testData = "string to be compressed"

func TestGzip(t *testing.T) {
	compressedData, err := compression.GzipString(testData)
	assert.NoError(t, err)
	originalData, err := compression.GunzipToString(compressedData)
	assert.NoError(t, err)
	assert.Equal(t, testData, originalData)
}
