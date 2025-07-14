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

	configKeyAuth, err := auth.NewConfigKeyAuthenticator(config, logger, auth.ProvideValueFromHeader("X-API-KEY"))
	if err != nil {
		return nil, fmt.Errorf("can not create config key auth: %w", err)
	}

	group := definitions.Group("/admin")
	group.Use(auth.NewChainHandler(map[string]auth.Authenticator{
		"api-key":    configKeyAuth,
		"basic-auth": basicAuth,
	}))

	group.GET("/authenticated", httpserver.CreateHandler(&AdminAuthenticatedHandler{}))

	// Add CRUD handlers and check for errors
	entityHandler := &MyEntityHandler{
		repo: &MyEntityRepository{},
	}

	if err := crud.AddCrudHandlers(config, logger, definitions, 0, "/myEntity", entityHandler); err != nil {
		return nil, fmt.Errorf("can not add crud handlers: %w", err)
	}

	return definitions, nil
}

func main() {
	application.Run(
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithModuleFactory("api", httpserver.New("default", apiDefiner)),
	)
}
