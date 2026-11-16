package stream_test

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"io"
	"testing"

	"github.com/justtrackio/gosoline/pkg/stream"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestCompressionZstd(t *testing.T) {
	for _, body := range [][]byte{
		[]byte("hello, world"),
		bytes.Repeat([]byte("compressible message "), 10000),
	} {
		compressed, err := stream.CompressMessage(stream.CompressionZstd, body)
		require.NoError(t, err)

		decompressed, err := stream.DecompressMessage(stream.CompressionZstd, compressed)
		require.NoError(t, err)
		require.Equal(t, body, decompressed)
	}

	// A frame emitted by an independent Zstandard implementation.
	frame, err := hex.DecodeString("28b52ffd0458a90000686973746f726963616c207a737464206672616d65223827b4")
	require.NoError(t, err)
	decompressed, err := stream.DecompressMessage(stream.CompressionZstd, frame)
	require.NoError(t, err)
	require.Equal(t, "historical zstd frame", string(decompressed))

	_, err = stream.DecompressMessage(stream.CompressionZstd, frame[:len(frame)-1])
	require.Error(t, err)
}
