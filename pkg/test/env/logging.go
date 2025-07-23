package env

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/log"
)

type (
	LoggerSettings struct {
		Level      string
		RecordLogs bool
	}

	RecordingLogger interface {
		log.GosoLogger
		Records() LogRecords
		Reset()
	}

	recordingLogger struct {
		log.GosoLogger
		mutex   *sync.RWMutex
		records LogRecords
	}

	ChannelRecords []LogRecord
	LogRecords     map[string]ChannelRecords

	LogRecord struct {
		Timestamp    time.Time
		Level        int
		Msg          string
		Args         []any
		Err          error
		Data         log.Data
		FormattedMsg string
	}

	handlerInMemoryWriter struct {
		level   int
		mutex   *sync.RWMutex
		records LogRecords
	}
)

func prepareLoggerSettings(options ...LoggerOption) (*LoggerSettings, error) {
	settings := &LoggerSettings{
		Level: log.LevelInfo,
	}

	for _, opt := range options {
		if err := opt(settings); err != nil {
			return nil, fmt.Errorf("can not apply option %T: %w", opt, err)
		}
	}

	return settings, nil
}

func NewRecordingConsoleLogger(config cfg.Config, options ...LoggerOption) (RecordingLogger, error) {
	settings, err := prepareLoggerSettings(options...)
	if err != nil {
		return nil, err
	}

	cl := clock.NewRealClock()
	handler := log.NewHandlerIoWriter(config, settings.Level, log.FormatterConsole, "test", "15:04:05.000", os.Stdout)

	logger := log.NewLoggerWithInterfaces(cl, []log.Handler{handler})

	recorder := recordingLogger{
		GosoLogger: logger,
		records:    make(LogRecords),
		mutex:      &sync.RWMutex{},
	}

	if settings.RecordLogs {
		err = logger.Option(log.WithHandlers(handlerInMemoryWriter{
			level:   log.LevelPriority(settings.Level),
			records: recorder.records,
			mutex:   recorder.mutex,
		}))
		if err != nil {
			return nil, fmt.Errorf("adding log recording handler to logger: %w", err)
		}
	}

	return recorder, nil
}

func (r recordingLogger) Records() LogRecords {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return funk.MergeMaps(r.records)
}

func (r recordingLogger) Reset() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for channel := range r.records {
		delete(r.records, channel)
	}
}

func (h handlerInMemoryWriter) ChannelLevel(string) (level *int, err error) {
	return nil, nil
}

func (h handlerInMemoryWriter) Level() int {
	return h.level
}

func (h handlerInMemoryWriter) Log(timestamp time.Time, level int, msg string, args []any, err error, data log.Data) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	channelRecords, ok := h.records[data.Channel]
	if !ok {
		channelRecords = make([]LogRecord, 0, 256)
	}

	h.records[data.Channel] = append(channelRecords, LogRecord{
		Timestamp:    timestamp,
		Data:         data,
		Level:        level,
		Msg:          msg,
		Args:         args,
		Err:          err,
		FormattedMsg: fmt.Sprintf(msg, args...),
	})

	return nil
}

func (logs ChannelRecords) Filter(condition func(LogRecord) bool) ChannelRecords {
	return funk.Filter(logs, condition)
}

func (logs LogRecords) Filter(condition func(LogRecord) bool) LogRecords {
	filtered := make(LogRecords)
	for channel, records := range logs {
		filteredRecords := funk.Filter(records, condition)
		if len(filteredRecords) > 0 {
			filtered[channel] = filteredRecords
		}
	}

	return filtered
}

func (logs LogRecords) Channel(channel string) ChannelRecords {
	return logs[channel]
}
