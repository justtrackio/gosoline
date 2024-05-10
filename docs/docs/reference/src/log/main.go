package main

import (
	"errors"
	"os"

	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	// will create an empty logger with a real clock and no handlers assigned
	logger := log.NewLogger()

	// create a handler which writes messages to stdout
	handler := log.NewHandlerIoWriter(
		// the min log level to write (trace, debug, info, warn, error)
		log.LevelDebug,
		// configuration of channels to filter for, if empty handler default will be used
		log.Channels{},
		// how to format the message. this will format in a console friendly way. log.FormatterJson would format log message as json
		log.FormatterConsole,
		// how to format the log time. uses the structure of the `time` package
		"15:04:05.000",
		// the io.Writer to write to. this case it's stdout
		os.Stdout,
	)

	// define logger options
	options := []log.Option{
		// add one or more handlers
		log.WithHandlers(handler),

		// add some fields which are added to every log message
		log.WithFields(map[string]any{
			"application": "gateway",
		}),
	}

	// apply options
	if err := logger.Option(options...); err != nil {
		panic(err)
	}

	// print message with different levels
	logger.Info("got an event with value %d", 42)
	logger.Warn("this can but shouldn't happen")
	logger.Error("we got an error: %s", errors.New("something bad"))
}
