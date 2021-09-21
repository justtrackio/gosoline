package test

import (
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/kernel"
)

func Application() kernel.Kernel {
	options := []application.Option{
		application.WithConfigFile("./config.dist.yml", "yml"),
	}

	return application.New(options...)
}
