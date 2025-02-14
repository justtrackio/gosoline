package log

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	AddHandlerFactory("iowriter", handlerIoWriterFactory)
}

type HandlerIoWriterSettings struct {
	Level           string   `cfg:"level" default:"info"`
	Channels        []string `cfg:"channels"`
	Formatter       string   `cfg:"formatter" default:"console"`
	TimestampFormat string   `cfg:"timestamp_format" default:"15:04:05.000"`
	Writer          string   `cfg:"writer" default:"stdout"`
}

func handlerIoWriterFactory(_ context.Context, config cfg.Config, name string) (Handler, error) {
	handlerConfigKey := getHandlerConfigKey(name)
	settings := &HandlerIoWriterSettings{}
	UnmarshalHandlerSettingsFromConfig(config, name, settings)

	var ok bool
	var err error
	var writerFactory IoWriterWriterFactory
	var writer io.Writer
	var formatter Formatter

	if writerFactory, ok = ioWriterFactories[settings.Writer]; !ok {
		return nil, fmt.Errorf("io writer of type %s not available", settings.Writer)
	}

	if writer, err = writerFactory(config, handlerConfigKey); err != nil {
		return nil, fmt.Errorf("can not create io writer of type %s: %w", settings.Writer, err)
	}

	if formatter, ok = formatters[settings.Formatter]; !ok {
		return nil, fmt.Errorf("io writer formatter of type %s not available", settings.Formatter)
	}

	return NewHandlerIoWriter(settings.Level, settings.Channels, formatter, settings.TimestampFormat, writer), nil
}

type handlerIoWriter struct {
	level           int
	channels        []string
	formatter       Formatter
	timestampFormat string
	writer          io.Writer
}

func NewHandlerIoWriter(level string, channels []string, formatter Formatter, timestampFormat string, writer io.Writer) *handlerIoWriter {
	return &handlerIoWriter{
		level:           LevelPriority(level),
		channels:        channels,
		formatter:       formatter,
		timestampFormat: timestampFormat,
		writer:          writer,
	}
}

func (h *handlerIoWriter) Channels() []string {
	return h.channels
}

func (h *handlerIoWriter) Level() int {
	return h.level
}

func (h *handlerIoWriter) Log(timestamp time.Time, level int, msg string, args []interface{}, logErr error, data Data) error {
	var err error
	var bytes []byte
	timestampStr := timestamp.Format(h.timestampFormat)

	if bytes, err = h.formatter(timestampStr, level, msg, args, logErr, data); err != nil {
		return fmt.Errorf("can not format log message: %w", err)
	}

	if _, err = h.writer.Write(bytes); err != nil {
		return fmt.Errorf("can not write log message: %w", err)
	}

	return nil
}
