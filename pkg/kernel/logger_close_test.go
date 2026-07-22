package kernel

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type closeTestLogger struct {
	closeErr   error
	closeCalls int
}

func (l *closeTestLogger) Debug(context.Context, string, ...any) {}
func (l *closeTestLogger) Info(context.Context, string, ...any)  {}
func (l *closeTestLogger) Warn(context.Context, string, ...any)  {}
func (l *closeTestLogger) Error(context.Context, string, ...any) {}

func (l *closeTestLogger) WithChannel(string) log.Logger {
	return l
}

func (l *closeTestLogger) WithFields(log.Fields) log.Logger {
	return l
}

func (l *closeTestLogger) Option(...log.Option) error {
	return nil
}

func (l *closeTestLogger) Close(context.Context) error {
	l.closeCalls++

	return l.closeErr
}

func TestKernelExitReportsLoggerCloseError(t *testing.T) {
	logger := &closeTestLogger{closeErr: errors.New("logger close failed")}
	exitCode := ExitCodeOk
	k := &kernel{
		ctx:      context.Background(),
		logger:   logger,
		exitCode: ExitCodeOk,
		exitHandler: func(code int) {
			exitCode = code
		},
	}

	output := captureStdout(t, k.exit)

	assert.Equal(t, 1, logger.closeCalls)
	assert.Equal(t, ExitCodeErr, exitCode)
	assert.Equal(t, "close logger completed with errors: logger close failed\n", output)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	stdout := os.Stdout
	os.Stdout = writer
	defer func() {
		os.Stdout = stdout
	}()

	fn()

	require.NoError(t, writer.Close())
	output, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	return string(output)
}
