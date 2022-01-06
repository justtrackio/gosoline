package test

import (
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/kernel"
)

func Application() kernel.Kernel {
	options := []application.Option{
		application.WithConfigFile("./config.dist.yml", "yml"),
		application.WithKernelExitHandler(func(code int) {}),
	}

	return application.New(options...)
}
