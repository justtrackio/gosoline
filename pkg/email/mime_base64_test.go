package email

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteMIMEBase64(t *testing.T) {
	data := bytes.Repeat([]byte("a"), 58)
	encoded := base64.StdEncoding.EncodeToString(data)
	want := encoded[:76] + mimeLineBreak + encoded[76:] + mimeLineBreak

	buffer := &bytes.Buffer{}
	require.NoError(t, writeMIMEBase64(buffer, data))
	require.Equal(t, want, buffer.String())

	buffer.Reset()
	require.NoError(t, writeMIMEBase64(buffer, nil))
	require.Empty(t, buffer.String())

	err := writeMIMEBase64(failingWriter{}, []byte("content"))
	require.ErrorIs(t, err, errMIMEWrite)

	err = writeMIMEBase64(lineBreakFailingWriter{}, bytes.Repeat([]byte("a"), 57))
	require.ErrorIs(t, err, errMIMEWrite)
}

var errMIMEWrite = errors.New("could not write MIME data")

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errMIMEWrite
}

type lineBreakFailingWriter struct{}

func (lineBreakFailingWriter) Write(data []byte) (int, error) {
	if string(data) == mimeLineBreak {
		return 0, errMIMEWrite
	}

	return len(data), nil
}

func decodeMIMEBase64(t testing.TB, reader io.Reader) []byte {
	t.Helper()

	decoded, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, reader))
	require.NoError(t, err)

	return decoded
}
