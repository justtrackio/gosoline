package cloud

import "context"

type FixedExecutor struct {
	out interface{}
	err error
}

func NewFixedExecutor(out interface{}, err error) RequestExecutor {
	return &FixedExecutor{
		out: out,
		err: err,
	}
}

func (e FixedExecutor) Execute(_ context.Context, f RequestFunction) (interface{}, error) {
	f()

	return e.out, e.err
}
