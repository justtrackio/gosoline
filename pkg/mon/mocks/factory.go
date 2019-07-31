package mocks

import "github.com/stretchr/testify/mock"

func NewLoggerMock(methods ...string) *Logger {
	logger := new(Logger)

	for _, m := range methods {
		logger.On(m, mock.Anything).Return(logger)
		logger.On(m, mock.Anything, mock.Anything).Return(logger)
		logger.On(m, mock.Anything, mock.Anything, mock.Anything).Return(logger)
		logger.On(m, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(logger)
	}

	return logger
}

func NewLoggerMockedAll() *Logger {
	return NewLoggerMock("Debug", "Debugf", "Info", "Infof", "Warn", "Warnf", "Error", "Errorf", "WithChannel", "WithContext", "WithFields")
}

func NewMetricWriterMockedAll() *MetricWriter {
	mw := new(MetricWriter)
	mw.On("Write", mock.Anything).Return()
	mw.On("WriteOne", mock.Anything).Return()

	return mw
}
