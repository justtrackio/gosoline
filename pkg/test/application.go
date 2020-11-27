package test

import (
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/kernel"
	"os"
	"path/filepath"
)

func Application() kernel.Kernel {
	ex, _ := os.Executable()
	configFilePath := filepath.Join(filepath.Dir(ex), "config.dist.yml")

	options := []application.Option{
		application.WithConfigFile(configFilePath, "yml"),
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
