package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/apiserver/auth"
	"github.com/applike/gosoline/pkg/apiserver/crud"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
)

type myTestStruct struct {
	Status string `json:"status"`
}

func apiDefiner(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
	definitions := &apiserver.Definitions{}

	definitions.GET("/json-from-map", apiserver.CreateHandler(&JsonResponseFromMapHandler{}))
	definitions.GET("/json-from-struct", apiserver.CreateHandler(&JsonResponseFromStructHandler{}))

	definitions.POST("/json-handler", apiserver.CreateJsonHandler(&JsonInputHandler{}))

	basicAuth, err := auth.NewBasicAuthAuthenticator(config, logger)
	if err != nil {
		return nil, fmt.Errorf("can not create basicAuth: %w", err)
	}

	group := definitions.Group("/admin")
	group.Use(auth.NewChainHandler(map[string]auth.Authenticator{
		"api-key":    auth.NewConfigKeyAuthenticator(config, logger, auth.ProvideValueFromHeader("X-API-KEY")),
		"basic-auth": basicAuth,
	}))

	group.GET("/authenticated", apiserver.CreateHandler(&AdminAuthenticatedHandler{}))

	crud.AddCrudHandlers(logger, definitions, 0, "/myEntity", &MyEntityHandler{
		repo: &MyEntityRepository{},
	})

	return definitions, nil
}

func main() {
	app := application.New(application.WithConfigFile("config.dist.yml", "yml"))
	app.Add("api", apiserver.New(apiDefiner))
	app.Run()
}
