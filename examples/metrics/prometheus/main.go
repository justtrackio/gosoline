package main

import (
	"context"

	"github.com/justtrackio/gosoline/examples/metrics/prometheus/counter"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

func apiDefiner(context.Context, cfg.Config, log.Logger) (*httpserver.Definitions, error) {
	definitions := &httpserver.Definitions{}

	ctrl := counter.NewCounterController()

	definitions.GET("/current-value", ctrl.Cur)
	definitions.GET("/increase", ctrl.Incr)
	definitions.GET("/decrease", ctrl.Decr)

	return definitions, nil
}

func main() {
	app := application.Default(
		application.WithModuleFactory("api", httpserver.New("default", apiDefiner)),
	)
	app.Run()
}
