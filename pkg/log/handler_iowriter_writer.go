package log

import (
	"io"
	"os"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

// IoWriterWriterFactory is a factory function for creating the underlying io.Writer for an iowriter handler.
type IoWriterWriterFactory func(config cfg.Config, configKey string) (io.Writer, error)

var ioWriterFactories = map[string]IoWriterWriterFactory{
	"stdout": ioWriterStdOutFactory,
}

// AddHandlerIoWriterFactory registers a new factory function for creating the underlying writer for an "iowriter" handler.
// This allows for extending the "iowriter" handler with custom write destinations.
func AddHandlerIoWriterFactory(typ string, factory IoWriterWriterFactory) {
	ioWriterFactories[typ] = factory
}

func ioWriterStdOutFactory(_ cfg.Config, _ string) (io.Writer, error) {
	return os.Stdout, nil
}
