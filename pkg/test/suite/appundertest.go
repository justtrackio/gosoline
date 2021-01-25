package suite

import "github.com/applike/gosoline/pkg/kernel"

type AppUnderTest interface {
	Stop()
	WaitDone()
}

type appUnderTest struct {
	kernel   kernel.Kernel
	waitDone func()
}

func newAppUnderTest(kernel kernel.Kernel, waitDone func()) *appUnderTest {
	return &appUnderTest{
		kernel:   kernel,
		waitDone: waitDone,
	}
}

func (a appUnderTest) Stop() {
	a.kernel.Stop("stopped by test")
}

func (a appUnderTest) WaitDone() {
	a.waitDone()
}
