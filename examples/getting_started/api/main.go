package main

import (
	"github.com/justtrackio/gosoline/examples/getting_started/api/definer"
	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/currency"
)

func main() {
	app := application.New(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithKernelSettingsFromConfig,
		application.WithLoggerHandlersFromConfig)

	app.Add("api", apiserver.New(definer.ApiDefiner))
	app.Add("currency", currency.NewCurrencyModule())

	app.Run()
}
