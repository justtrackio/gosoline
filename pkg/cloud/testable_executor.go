package cloud

import "context"

type TestExecution struct {
	Output interface{}
	Err    error
}

type TestableExecutor struct {
	executions []TestExecution
	current    int
}

func NewTestableExecutor(executions []TestExecution) *TestableExecutor {
	return &TestableExecutor{
		executions: executions,
	}
}

func (t *TestableExecutor) Execute(_ context.Context, f RequestFunction) (interface{}, error) {
	f()

	c := t.current
	t.current++

	return t.executions[c].Output, t.executions[c].Err
}
