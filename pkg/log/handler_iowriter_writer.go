package log

import (
	"io"
	"os"

	"github.com/applike/gosoline/pkg/cfg"
)

type IoWriterWriterFactory func(config cfg.Config, configKey string) (io.Writer, error)

var ioWriterFactories = map[string]IoWriterWriterFactory{
	"stdout": ioWriterStdOutFactory,
}

func AddHandlerIoWriterFactory(typ string, factory IoWriterWriterFactory) {
	ioWriterFactories[typ] = factory
}

func ioWriterStdOutFactory(_ cfg.Config, _ string) (io.Writer, error) {
	return os.Stdout, nil
}
