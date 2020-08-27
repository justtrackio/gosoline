package aws

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

type TestExecution struct {
	Output interface{}
	Err    error
}

type TestableExecutor struct {
	clientMock *mock.Mock
	executions []TestExecution
	current    int
}

func NewTestableExecutor(clientMock *mock.Mock, executions ...TestExecution) *TestableExecutor {
	return &TestableExecutor{
		clientMock: clientMock,
		executions: executions,
	}
}

func (e *TestableExecutor) Execute(_ context.Context, f RequestFunction) (interface{}, error) {
	f()

	c := e.current
	e.current++

	if c >= len(e.executions) {
		panic("there is no available test execution")
	}

	return e.executions[c].Output, e.executions[c].Err
}

func (e *TestableExecutor) ExpectExecution(clientMethod string, input interface{}, output interface{}, err error) {
	e.clientMock.On(clientMethod, input).Return(nil, output)

	e.executions = append(e.executions, TestExecution{
		Output: output,
		Err:    err,
	})
}

func (e *TestableExecutor) AssertExpectations(t *testing.T) {
	e.clientMock.AssertExpectations(t)

	if e.current == len(e.executions) {
		return
	}

	assert.Fail(t, "not all executions have been executed")
}
