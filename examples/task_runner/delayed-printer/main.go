package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/cfg"
	taskRunner "github.com/justtrackio/gosoline/pkg/conc/task_runner"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type delayedPrintHandler struct {
	logger log.Logger
}

type delayedPrintHandlerInput struct {
	Delay   int    `json:"delay"`
	Message string `json:"message"`
}

func (d delayedPrintHandler) GetInput() any {
	return &delayedPrintHandlerInput{}
}

func (d delayedPrintHandler) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	input := request.Body.(*delayedPrintHandlerInput)

	err := taskRunner.RunTask(ctx, kernel.NewModuleFunc(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(input.Delay) * time.Second):
		}

		d.logger.Info(ctx, "printing delayed message: %s", input.Message)

		return nil
	}))
	if err != nil {
		d.logger.Error(ctx, "failed to run task: %w", err)

		return httpserver.NewStatusResponse(http.StatusInternalServerError), nil
	}

	return httpserver.NewStatusResponse(http.StatusNoContent), nil
}

func newDelayedPrintHandler(logger log.Logger) (gin.HandlerFunc, error) {
	return httpserver.CreateJsonHandler(delayedPrintHandler{
		logger: logger.WithChannel("delayed-print-handler"),
	}), nil
}

func apiDefiner(_ context.Context, _ cfg.Config, logger log.Logger) (*httpserver.Definitions, error) {
	d := &httpserver.Definitions{}

	delayedPrintHandler, err := newDelayedPrintHandler(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create delayed print handler: %w", err)
	}

	d.POST("/delayed-print", delayedPrintHandler)

	return d, nil
}

func main() {
	application.RunHttpServers(map[string]httpserver.Definer{
		"default": apiDefiner,
	})
}
