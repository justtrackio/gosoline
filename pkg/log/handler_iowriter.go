package log

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

func init() {
	AddHandlerFactory("iowriter", handlerIoWriterFactory)
}

type HandlerIoWriterSettings struct {
	Level           string `cfg:"level" default:"info"`
	Formatter       string `cfg:"formatter" default:"console"`
	TimestampFormat string `cfg:"timestamp_format" default:"15:04:05.000"`
	Writer          string `cfg:"writer" default:"stdout"`
}

type ChannelSetting struct {
	Level string `cfg:"level"`
}

func handlerIoWriterFactory(config cfg.Config, name string) (Handler, error) {
	handlerConfigKey := getHandlerConfigKey(name)

	settings := &HandlerIoWriterSettings{}
	if err := UnmarshalHandlerSettingsFromConfig(config, name, settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal handler settings: %w", err)
	}

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

	return NewHandlerIoWriter(config, settings.Level, formatter, name, settings.TimestampFormat, writer), nil
}

type handlerIoWriter struct {
	config          cfg.Config
	lck             sync.RWMutex
	level           int
	channels        map[string]*int
	formatter       Formatter
	name            string
	timestampFormat string
	writer          io.Writer
}

func NewHandlerIoWriter(config cfg.Config, level string, formatter Formatter, name string, timestampFormat string, writer io.Writer) Handler {
	return &handlerIoWriter{
		config:          config,
		level:           LevelPriority(level),
		channels:        make(map[string]*int),
		formatter:       formatter,
		name:            name,
		timestampFormat: timestampFormat,
		writer:          writer,
	}
}

func (h *handlerIoWriter) ChannelLevel(name string) (level *int, err error) {
	h.lck.RLock()
	cached, ok := h.channels[name]
	h.lck.RUnlock()

	if ok {
		return cached, nil
	}

	h.lck.Lock()
	defer h.lck.Unlock()

	key := fmt.Sprintf("%s.channels.%s", getHandlerConfigKey(h.name), name)
	settings := &ChannelSetting{}
	err = h.config.UnmarshalKey(key, settings)
	if err != nil {
		// store that we don't have a setting to avoid spamming errors
		h.channels[name] = nil

		return nil, fmt.Errorf("can not unmarshal channel settings: %w", err)
	}

	priority := LevelPriority(settings.Level)
	h.channels[name] = &priority

	return &priority, nil
}

func (h *handlerIoWriter) Level() int {
	return h.level
}

func (h *handlerIoWriter) Log(timestamp time.Time, level int, msg string, args []any, logErr error, data Data) error {
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
