package mon

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/clock"
	"os"
)

const stdoutLoggerType = "stdout"

type stdoutLoggerHandlerSettings struct {
	BaseLoggerHandlerSettings
}

func init() {
	HandlerFactories[stdoutLoggerType] = newStdoutLoggerHandler
}

func newStdoutLoggerHandler(config cfg.Config, name string) (Handler, error) {
	settings := stdoutLoggerHandlerSettings{}
	config.UnmarshalKey(fmt.Sprintf("mon.logger.handler.%s", name), &settings)

	return NewIowriterLoggerHandler(clock.NewRealClock(), settings.OutputFormat, os.Stdout, settings.TimestampFormat, settings.LogLevels)
}
