package log

import (
	"os"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
)

// NewCliLogger creates a logger suitable for CLI applications, writing to stdout.
func NewCliLogger() Logger {
	handler := NewCliHandler()

	return NewLoggerWithInterfaces(clock.Provider, []Handler{handler})
}

// NewCliHandler creates a handler configured for CLI output (console format, info level, stdout).
func NewCliHandler() Handler {
	return NewHandlerIoWriter(cfg.New(), PriorityInfo, FormatterConsole, "cli", "15:04:05.000", os.Stdout)
}
