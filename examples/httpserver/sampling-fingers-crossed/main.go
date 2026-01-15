package main

import (
	"context"
	_ "embed"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

func main() {
	application.RunHttpDefaultServer(
		func(ctx context.Context, config cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
			definitions := &httpserver.Definitions{}

			definitions.GET("/success", httpserver.CreateHandler(&successHandler{logger: logger}))
			definitions.GET("/fail", httpserver.CreateHandler(&failHandler{logger: logger}))

			return definitions, nil
		},
		application.WithConfigFile("config.dist.yml", "yml"),
		application.WithSampling,
	)
}

type successHandler struct {
	logger log.Logger
}

func (h *successHandler) Handle(ctx context.Context, _ *httpserver.Request) (*httpserver.Response, error) {
	h.logger.Info(ctx, "some log line before succeeding")

	return httpserver.NewJsonResponse(map[string]any{"Status": "ok"}), nil
}

type failHandler struct {
	logger log.Logger
}

func (h *failHandler) Handle(ctx context.Context, _ *httpserver.Request) (*httpserver.Response, error) {
	h.logger.Info(ctx, "some log line before failing")

	return httpserver.NewJsonResponse(map[string]any{"Error": "something went wrong"}, httpserver.WithStatusCode(http.StatusInternalServerError)), nil
}
