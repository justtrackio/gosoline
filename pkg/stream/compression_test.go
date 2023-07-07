package stream_test

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
)

func TestCompressionNone(t *testing.T) {
	for _, body := range []string{
		"",
		"\000",
		"hello, world",
		"this message contains special characters: ä, 💩, 猫",
	} {
		compressed, err := stream.CompressMessage(stream.CompressionNone, []byte(body))
		assert.NoError(t, err)
		assert.Equal(t, body, string(compressed))

		decompressed, err := stream.DecompressMessage(stream.CompressionNone, compressed)
		assert.NoError(t, err)
		assert.Equal(t, body, string(decompressed))
	}
}

func TestCompressionGzip(t *testing.T) {
	for _, body := range []string{
		"",
		"\000",
		"hello, world",
		"this message contains special characters: ä, 💩, 猫",
		"loren ipsum and so on, this text goes on and on. loren ipsum and so on, this text goes on and on. loren ipsum and so on, this text goes on and on. loren ipsum and so on, this text goes on and on. loren ipsum and so on, this text goes on and on. loren ipsum and so on, this text goes on and on. ",
	} {
		compressed, err := stream.CompressMessage(stream.CompressionGZip, []byte(body))
		assert.NoError(t, err)
		// for large messages, it should actually reduce their size
		if len(body) > 100 {
			assert.Less(t, len(compressed), len(body))
		}

		reader, err := gzip.NewReader(bytes.NewReader(compressed))
		assert.NoError(t, err)
		decompressedBody, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, body, string(decompressedBody))

		decompressed, err := stream.DecompressMessage(stream.CompressionGZip, compressed)
		assert.NoError(t, err)
		assert.Equal(t, body, string(decompressed))
	}
}
