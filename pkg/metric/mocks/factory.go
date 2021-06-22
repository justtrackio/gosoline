package mocks

import (
	"github.com/applike/gosoline/pkg/metric"
	"github.com/stretchr/testify/mock"
)

func NewWriterMockedAll() *Writer {
	mw := new(Writer)
	mw.On("GetPriority").Return(metric.PriorityLow).Maybe()
	mw.On("Write", mock.AnythingOfType("metric.Data")).Return().Maybe()
	mw.On("WriteOne", mock.AnythingOfType("*metric.Datum")).Return().Maybe()

	return mw
}
