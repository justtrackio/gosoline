package mocks

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/stretchr/testify/mock"
)

const defaultBufferSize = 100

type LogRecord struct {
	Timestamp time.Time
	Level     string
	Template  string
	Arguments []any
}

func (r LogRecord) String() string {
	return fmt.Sprintf("%s [%s] %s", r.Timestamp.Format("2006-01-02T15:04:05.999Z07:00"), r.Level, fmt.Sprintf(r.Template, r.Arguments...))
}

type LoggerMock struct {
	Logger
	buffer     []LogRecord
	bufferLck  sync.Mutex
	bufferSize int
}

func (l *LoggerMock) WithBufferSize(bufferSize int) *LoggerMock {
	l.bufferLck.Lock()
	defer l.bufferLck.Unlock()

	l.bufferSize = bufferSize

	return l
}

func (l *LoggerMock) PrintBufferedLogsOnFailure(t *testing.T) {
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}

		logs := l.BufferedLogs()

		fmt.Printf("Last %d log messages:\n", len(logs))
		for _, msg := range logs {
			fmt.Println(msg.String())
		}
	})
}

func (l *LoggerMock) BufferedLogs() []LogRecord {
	l.bufferLck.Lock()
	defer l.bufferLck.Unlock()

	if len(l.buffer) > l.bufferSize {
		return append([]LogRecord{}, l.buffer[len(l.buffer)-l.bufferSize:]...)
	}

	return append([]LogRecord{}, l.buffer...)
}

func NewLoggerMock() *LoggerMock {
	logger := new(LoggerMock)

	logger.On("WithChannel", mock.AnythingOfType("string")).Return(logger).Maybe()
	logger.On("WithContext", mock.Anything).Return(logger).Maybe()
	logger.On("WithFields", mock.Anything).Return(logger).Maybe()

	return logger
}

func NewLoggerMockedAll() *LoggerMock {
	return NewLoggerMockedUntilLevel(log.PriorityError)
}

// NewLoggerMockedUntilLevel returns a logger mocked up to the given log level. All other calls will cause an error and fail the test.
func NewLoggerMockedUntilLevel(level int) *LoggerMock {
	logger := NewLoggerMock()

	mockLoggerMethod(logger, "Debug", log.LevelDebug, level >= log.PriorityDebug)
	mockLoggerMethod(logger, "Info", log.LevelInfo, level >= log.PriorityInfo)
	mockLoggerMethod(logger, "Warn", log.LevelWarn, level >= log.PriorityWarn)
	mockLoggerMethod(logger, "Error", log.LevelError, level >= log.PriorityError)

	return logger
}

func mockLoggerMethod(logger *LoggerMock, method string, level string, allowed bool) {
	anythings := make(mock.Arguments, 0)
	f := inspectLogFunction(logger, level, allowed)

	for i := 0; i < 10; i++ {
		anythings = append(anythings, mock.Anything)
		logger.On(method, anythings...).Run(f).Return(logger).Maybe()
	}
}

func inspectLogFunction(logger *LoggerMock, level string, allowed bool) func(args mock.Arguments) {
	return func(args mock.Arguments) {
		if !allowed {
			panic(fmt.Errorf("invalid log message '%s'. Logs of level %s are not allowed", args.Get(0), level))
		}

		logger.bufferLck.Lock()
		defer logger.bufferLck.Unlock()

		if logger.bufferSize == 0 {
			logger.bufferSize = defaultBufferSize
		}

		logger.buffer = append(logger.buffer, LogRecord{
			Timestamp: time.Now(),
			Level:     level,
			Template:  args.Get(0).(string),
			Arguments: args[1:],
		})
		if len(logger.buffer) >= logger.bufferSize*2 {
			copy(logger.buffer[0:logger.bufferSize], logger.buffer[logger.bufferSize:logger.bufferSize*2])
			logger.buffer = logger.buffer[:logger.bufferSize]
		}
	}
}
