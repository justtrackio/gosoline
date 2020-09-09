package mon

import (
	"fmt"
	"github.com/applike/gosoline/pkg/clock"
	"github.com/thoas/go-funk"
	"io"
	"os"
)

type BaseLoggerHandlerSettings struct {
	LogLevels       []string `cfg:"levels" default:"trace,debug,info,warn,error,fatal,panic"`
	OutputFormat    string   `cfg:"outputFormat" default:"console"`
	TimestampFormat string   `cfg:"timestampFormat" default:"2006-01-02T15:04:05Z07:00"`
}

func NewIowriterLoggerHandler(clock clock.Clock, format string, output io.Writer, timestampFormat string, logLevels []string) (Handler, error) {
	return &iowriterLogger{
		clock:           clock,
		formatter:       formatters[format],
		logLevels:       logLevels,
		output:          output,
		timestampFormat: timestampFormat,
	}, nil
}

type iowriterLogger struct {
	clock           clock.Clock
	formatter       formatter
	logLevels       []string
	output          io.Writer
	timestampFormat string
}

func (s *iowriterLogger) Write(level string, msg string, logErr error, metadata Metadata) {
	if !funk.ContainsString(s.logLevels, level) {
		return
	}

	timestamp := s.clock.Now().Format(s.timestampFormat)

	buffer, err := s.formatter(timestamp, level, msg, logErr, &metadata)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to format log, %v\n", err)
	}

	_, err = s.output.Write(buffer)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to write to log, %v\n", err)
	}
}
