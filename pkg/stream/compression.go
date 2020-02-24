package stream

import (
	"bytes"
	"compress/gzip"
	"fmt"
)

const (
	CompressionNone = "none"
	CompressionGZip = "application/gzip"
)

type MessageBodyCompressor interface {
	Compress(body []byte) ([]byte, error)
	Decompress(body []byte) ([]byte, error)
}

var messageBodyCompressors = map[string]MessageBodyCompressor{
	CompressionNone: new(noopCompressor),
	CompressionGZip: new(gZipCompressor),
}

type noopCompressor struct {
}

func (n noopCompressor) Compress(body []byte) ([]byte, error) {
	return body, nil
}

func (n noopCompressor) Decompress(body []byte) ([]byte, error) {
	return body, nil
}

type gZipCompressor struct {
}

func (g gZipCompressor) Compress(body []byte) ([]byte, error) {
	if body == nil {
		return body, nil
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
		return body, nil
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
