package env

import (
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/applike/gosoline/pkg/log"
	"os"
)

type LoggerSettings struct {
	Level string
}

func NewConsoleLogger(options ...LoggerOption) (log.GosoLogger, error) {
	settings := &LoggerSettings{
		Level: log.LevelInfo,
	}

	for _, opt := range options {
		if err := opt(settings); err != nil {
			return nil, fmt.Errorf("can not apply option %T: %w", opt, err)
		}
	}

	cl := clock.NewRealClock()
	handler := log.NewHandlerIoWriter(log.LevelInfo, []string{}, log.FormatterConsole, "15:04:05.000", os.Stdout)

	return log.NewLoggerWithInterfaces(cl, []log.Handler{handler}), nil
}
