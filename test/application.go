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
