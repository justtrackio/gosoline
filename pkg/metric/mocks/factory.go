package mocks

import (
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/stretchr/testify/mock"
)

func NewWriterMockedAll() *Writer {
	mw := new(Writer)
	mw.On("GetPriority").Return(metric.PriorityLow).Maybe()
	mw.On("Write", matcher.Context, mock.AnythingOfType("metric.Data")).Return().Maybe()
	mw.On("WriteOne", matcher.Context, mock.AnythingOfType("*metric.Datum")).Return().Maybe()

	return mw
}
