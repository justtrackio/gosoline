package test

import (
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/kernel"
)

func Application() kernel.Kernel {
	options := []application.Option{
		application.WithConfigFile("./config.dist.yml", "yml"),
	}

	return application.New(options...)
}

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
