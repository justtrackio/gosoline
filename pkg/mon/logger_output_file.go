package mon

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"os"
)

const fileLoggerType = "file"

type fileLoggerHandlerSettings struct {
	BaseLoggerHandlerSettings
	Filename string `cfg:"filename"`
}

func init() {
	HandlerFactories[fileLoggerType] = newFileLoggerHandler
}

func newFileLoggerHandler(config cfg.Config, name string) (Handler, error) {
	settings := fileLoggerHandlerSettings{}
	config.UnmarshalKey(fmt.Sprintf("mon.logger.handler.%s", name), &settings)

	outputFile, err := os.OpenFile(settings.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file '%s': %w", settings.Filename, err)
	}

	return NewIowriterLoggerHandler(clock.NewRealClock(), settings.OutputFormat, outputFile, settings.TimestampFormat, settings.LogLevels)
}
