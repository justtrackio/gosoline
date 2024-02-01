package main

import (
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/currency"
	"github.com/justtrackio/gosoline/pkg/httpserver"
)

func main() {
	application.Run(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithLoggerHandlersFromConfig,

		application.WithModuleFactory("api", httpserver.New("default", ApiDefiner)),
		application.WithModuleFactory("currency", currency.NewCurrencyModule()),
	)
}
