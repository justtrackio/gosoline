package suite

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/kernel"
)

type AppUnderTest interface {
	Context() context.Context
	Stop()
	WaitDone()
}

type appUnderTest struct {
	ctx      context.Context
	kernel   kernel.Kernel
	waitDone func()
}

func newAppUnderTest(ctx context.Context, kernel kernel.Kernel, waitDone func()) *appUnderTest {
	return &appUnderTest{
		ctx:      ctx,
		kernel:   kernel,
		waitDone: waitDone,
	}
}

func (a appUnderTest) Context() context.Context {
	return a.ctx
}

func (a appUnderTest) Stop() {
	a.kernel.Stop("stopped by test")
}

func (a appUnderTest) WaitDone() {
	a.waitDone()
}
