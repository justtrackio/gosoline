package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/auth"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	"github.com/justtrackio/gosoline/pkg/log"
)

type myTestStruct struct {
	Status string `json:"status"`
}

func apiDefiner(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
	definitions := &httpserver.Definitions{}

	definitions.GET("/json-from-map", httpserver.CreateHandler(&JsonResponseFromMapHandler{}))
	definitions.GET("/json-from-struct", httpserver.CreateHandler(&JsonResponseFromStructHandler{}))

	definitions.POST("/json-handler", httpserver.CreateJsonHandler(&JsonInputHandler{}))

	basicAuth, err := auth.NewBasicAuthAuthenticator(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create basicAuth: %w", err)
	}

	group := definitions.Group("/admin")
	group.Use(auth.NewChainHandler(map[string]auth.Authenticator{
		"api-key":    auth.NewConfigKeyAuthenticator(config, logger, auth.ProvideValueFromHeader("X-API-KEY")),
		"basic-auth": basicAuth,
	}))

	group.GET("/authenticated", httpserver.CreateHandler(&AdminAuthenticatedHandler{}))

	crud.AddCrudHandlers(config, logger, definitions, 0, "/myEntity", &MyEntityHandler{
		repo: &MyEntityRepository{},
	})

	return definitions, nil
}

func main() {
	application.Run(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithModuleFactory("api", httpserver.New("default", apiDefiner)),
	)
}
