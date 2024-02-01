package httpserver_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/encoding/base64"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/stretchr/testify/assert"
)

func TestCountingBodyReader(t *testing.T) {
	r, readBytes := httpserver.NewCountingBodyReader(io.NopCloser(bytes.NewReader([]byte{1, 2, 3, 4})))

	data, err := io.ReadAll(r)
	assert.NoError(t, err)
	assert.Equal(t, data, []byte{1, 2, 3, 4})
	assert.Equal(t, *readBytes, 4)
}

type responseWriter struct {
	gin.ResponseWriter
	written []any
}

func (r *responseWriter) Write(p []byte) (int, error) {
	r.written = append(r.written, p)

	return len(p), nil
}

func (r *responseWriter) WriteString(s string) (int, error) {
	r.written = append(r.written, s)

	return len(s), nil
}

func TestCountingBodyWriter(t *testing.T) {
	rw := &responseWriter{}
	w, writtenBytes := httpserver.NewCountingBodyWriter(rw)

	n, err := w.Write([]byte{1, 2, 3, 4})
	assert.NoError(t, err)
	assert.Equal(t, n, 4)
	assert.Equal(t, *writtenBytes, 4)

	n, err = w.WriteString("abcde")
	assert.NoError(t, err)
	assert.Equal(t, n, 5)
	assert.Equal(t, *writtenBytes, 9)

	assert.Equal(t, rw.written, []any{
		[]byte{1, 2, 3, 4},
		"abcde",
	})
}

func TestGZipBodyReader(t *testing.T) {
	compressedBody, err := base64.DecodeString("H4sIAAAAAAACAwvJyCxWAKJEheLE3IKcVIWS1IoShcS8FIXMEoXijPzyYgUggSadnJinUFCUX5aZkoqQScsvAnJA8noAgzgFIFUAAAA=")
	assert.NoError(t, err)

	reader, readUncompressedBytes, err := httpserver.NewGZipBodyReader(io.NopCloser(bytes.NewReader(compressedBody)))
	assert.NoError(t, err)

	read, err := io.ReadAll(reader)
	assert.NoError(t, err)
	assert.Equal(t, []byte("This is a sample text and it shows how a sample text can provide a sample for a text."), read)
	assert.Equal(t, 85, *readUncompressedBytes)
}
