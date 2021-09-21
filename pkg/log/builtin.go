package log

import (
	"os"

	"github.com/justtrackio/gosoline/pkg/clock"
)

func NewCliLogger() Logger {
	handler := NewCliHandler()

	return NewLoggerWithInterfaces(clock.Provider, []Handler{handler})
}

func NewCliHandler() Handler {
	return NewHandlerIoWriter(LevelInfo, []string{}, FormatterConsole, "15:04:05.000", os.Stdout)
}
