package db_repo

import (
	"context"
	"fmt"
	"net/http"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type ModelHandler interface{}

type HttpHandler struct {
	modelHandlers map[string]ModelHandler
}

func NewHttpHandler(_ context.Context, config cfg.Config, logger log.Logger) (*HttpHandler, error) {
	return &HttpHandler{}, nil
}

func (h *HttpHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	fmt.Println("got it")
}
