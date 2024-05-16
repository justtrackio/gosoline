package env

import (
	"fmt"
	"os"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type (
	LoggerSettings struct {
		Level      string
		RecordLogs bool
	}

	RecordingLogger interface {
		log.GosoLogger
		Records() []LogRecord
	}

	recordingLogger struct {
		log.GosoLogger
		records *mapx.MapX
	}

	LogRecords []LogRecord

	LogRecord struct {
		Timestamp    time.Time
		Level        int
		Msg          string
		Args         []interface{}
		Err          error
		Data         log.Data
		FormattedMsg string
	}

	handlerInMemoryWriter struct {
		level   int
		records *mapx.MapX
		uuid.Uuid
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

func NewRecordingConsoleLogger(options ...LoggerOption) (RecordingLogger, error) {
	settings, err := prepareLoggerSettings(options...)
	if err != nil {
		return nil, err
	}

	cl := clock.NewRealClock()
	handler := log.NewHandlerIoWriter(settings.Level, []string{}, log.FormatterConsole, "15:04:05.000", os.Stdout)

	logger := log.NewLoggerWithInterfaces(cl, []log.Handler{handler})

	recorder := recordingLogger{
		GosoLogger: logger,
		records:    mapx.NewMapX(),
	}

	if settings.RecordLogs {
		err = logger.Option(log.WithHandlers(handlerInMemoryWriter{
			level:   log.LevelPriority(settings.Level),
			records: recorder.records,
			Uuid:    uuid.New(),
		}))
		if err != nil {
			return nil, fmt.Errorf("adding log recording handler to logger: %w", err)
		}
	}

	return recorder, nil
}

func (r recordingLogger) Records() []LogRecord {
	items := r.records.Msi()

	records := make([]LogRecord, 0, len(items))
	for _, record := range items {
		records = append(records, record.(LogRecord))
	}

	return records
}

func (h handlerInMemoryWriter) Channels() []string {
	return []string{}
}

func (h handlerInMemoryWriter) Level() int {
	return h.level
}

func (h handlerInMemoryWriter) Log(timestamp time.Time, level int, msg string, args []interface{}, err error, data log.Data) error {
	h.records.Set(h.NewV4(), LogRecord{
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

func (logs LogRecords) Filter(condition func(LogRecord) bool) LogRecords {
	out := make(LogRecords, 0)

	for _, log := range logs {
		if condition(log) {
			out = append(out, log)
		}
	}

	return out
}
