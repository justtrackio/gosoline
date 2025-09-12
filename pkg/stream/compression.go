package stream

import (
	"bytes"
	"compress/gzip"
	"fmt"

	"github.com/golang/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
)

type CompressionType string

const (
	CompressionNone   CompressionType = "none"
	CompressionGZip   CompressionType = "application/gzip"
	CompressionSnappy CompressionType = "application/snappy"
	CompressionLZ4    CompressionType = "application/lz4"
	CompressionZstd   CompressionType = "application/zstd"
)

func (s CompressionType) String() string {
	return string(s)
}

var _ fmt.Stringer = CompressionType("")

type MessageBodyCompressor interface {
	Compress(body []byte) ([]byte, error)
	Decompress(body []byte) ([]byte, error)
}

var messageBodyCompressors = map[CompressionType]MessageBodyCompressor{
	CompressionNone:   new(noopCompressor),
	CompressionGZip:   new(gZipCompressor),
	CompressionSnappy: new(snappyCompressor),
	CompressionLZ4:    new(lz4Compressor),
	CompressionZstd:   new(zstdCompressor),
}

func CompressMessage(compression CompressionType, body []byte) ([]byte, error) {
	compressor, ok := messageBodyCompressors[compression]

	if !ok {
		return nil, fmt.Errorf("there is no compressor for compression '%s'", compression)
	}

	compressed, err := compressor.Compress(body)
	if err != nil {
		return nil, fmt.Errorf("failed to compress message body: %w", err)
	}

	return compressed, nil
}

func DecompressMessage(compression CompressionType, body []byte) ([]byte, error) {
	compressor, ok := messageBodyCompressors[compression]

	if !ok {
		return nil, fmt.Errorf("there is no decompressor for compression '%s'", compression)
	}

	decompressed, err := compressor.Decompress(body)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress message body: %w", err)
	}

	return decompressed, nil
}

type noopCompressor struct{}

func (n noopCompressor) Compress(body []byte) ([]byte, error) {
	return body, nil
}

func (n noopCompressor) Decompress(body []byte) ([]byte, error) {
	return body, nil
}

type gZipCompressor struct{}

func (g gZipCompressor) Compress(body []byte) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	var out bytes.Buffer
	zw := gzip.NewWriter(&out)

	if _, err := zw.Write(body); err != nil {
		return nil, fmt.Errorf("can not write body to gzip: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("can not close gzip writer: %w", err)
	}

	compressed := out.Bytes()

	return compressed, nil
}

func (g gZipCompressor) Decompress(body []byte) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	bufBody := bytes.NewBuffer(body)
	bufOut := &bytes.Buffer{}

	reader, err := gzip.NewReader(bufBody)
	if err != nil {
		return nil, fmt.Errorf("can not create gzip reader from body: %w", err)
	}

	if _, err := bufOut.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("can not read from gzip reader: %w", err)
	}

	uncompressed := bufOut.Bytes()

	return uncompressed, nil
}

type snappyCompressor struct{}

func (g snappyCompressor) Compress(body []byte) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	var out bytes.Buffer
	zw := snappy.NewBufferedWriter(&out)

	if _, err := zw.Write(body); err != nil {
		return nil, fmt.Errorf("can not write body to snappy: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("can not close snappy writer: %w", err)
	}

	compressed := out.Bytes()

	return compressed, nil
}

func (g snappyCompressor) Decompress(body []byte) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	bufBody := bytes.NewBuffer(body)
	bufOut := &bytes.Buffer{}

	reader := snappy.NewReader(bufBody)

	if _, err := bufOut.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("can not read from snappy reader: %w", err)
	}

	uncompressed := bufOut.Bytes()

	return uncompressed, nil
}

type lz4Compressor struct{}

func (g lz4Compressor) Compress(body []byte) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	var out bytes.Buffer
	zw := lz4.NewWriter(&out)

	if _, err := zw.Write(body); err != nil {
		return nil, fmt.Errorf("can not write body to lz4: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("can not close lz4 writer: %w", err)
	}

	compressed := out.Bytes()

	return compressed, nil
}

func (g lz4Compressor) Decompress(body []byte) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	bufBody := bytes.NewBuffer(body)
	bufOut := &bytes.Buffer{}

	reader := lz4.NewReader(bufBody)

	if _, err := bufOut.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("can not read from lz4 reader: %w", err)
	}

	uncompressed := bufOut.Bytes()

	return uncompressed, nil
}

type zstdCompressor struct{}

func (g zstdCompressor) Compress(body []byte) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	var out bytes.Buffer
	zw, err := zstd.NewWriter(&out)
	if err != nil {
		return nil, fmt.Errorf("can not create zstd writer: %w", err)
	}

	if _, err := zw.Write(body); err != nil {
		return nil, fmt.Errorf("can not write body to zstd: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("can not close zstd writer: %w", err)
	}

	compressed := out.Bytes()

	return compressed, nil
}

func (g zstdCompressor) Decompress(body []byte) ([]byte, error) {
	if body == nil {
		return nil, nil
	}

	bufBody := bytes.NewBuffer(body)
	bufOut := &bytes.Buffer{}

	reader, err := zstd.NewReader(bufBody)
	if err != nil {
		return nil, fmt.Errorf("can not create zstd reader: %w", err)
	}

	if _, err := bufOut.ReadFrom(reader); err != nil {
		return nil, fmt.Errorf("can not read from zstd reader: %w", err)
	}

	uncompressed := bufOut.Bytes()

	return uncompressed, nil
}
