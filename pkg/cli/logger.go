package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
)

func newCliLogger(config cfg.Config) (log.Logger, error) {
	var err error
	var writer io.Writer

	if writer, err = log.NewIoWriterFile("logs.log"); err != nil {
		return nil, fmt.Errorf("can not create io file writer for logger: %w", err)
	}

	handler := log.NewHandlerIoWriter(config, log.PriorityInfo, log.FormatterConsole, "cli", "", writer)
	logger := log.NewLoggerWithInterfaces(clock.Provider, []log.Handler{handler})

	return logger, nil
}

type LogHandler struct {
}

func (l LogHandler) ChannelLevel(name string) (level *int, err error) {
	return nil, nil
}

func (l LogHandler) Level() int {
	return log.PriorityDebug
}

func (l LogHandler) Log(ctx context.Context, timestamp time.Time, level int, msg string, args []any, logErr error, data log.Data) error {
	var err error
	var bytes []byte
	timestampStr := timestamp.Format("15:04:05.000")

	if bytes, err = log.FormatterConsole(timestampStr, level, msg, args, logErr, data); err != nil {
		return fmt.Errorf("can not format log message: %w", err)
	}

	if _, err = os.Stdout.Write(bytes); err != nil {
		return fmt.Errorf("can not write log message: %w", err)
	}

	return nil
}
