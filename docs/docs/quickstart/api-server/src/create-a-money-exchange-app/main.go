package main

import (
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/currency"
)

func main() {
	application.Run(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithLoggerHandlersFromConfig,

		application.WithModuleFactory("api", apiserver.New(ApiDefiner)),
		application.WithModuleFactory("currency", currency.NewCurrencyModule()),
	)
}
