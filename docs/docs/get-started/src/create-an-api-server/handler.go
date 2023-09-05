package main

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type Todo struct {
	Id        int    `form:"id"`
	Text      string `form:"text"`
	CreatedAt time.Time
}

type TodoHandler struct {
	logger log.Logger
}

func (t TodoHandler) GetInput() interface{} {
	return &Todo{}
}

func NewTodoHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*TodoHandler, error) {
	handler := &TodoHandler{
		logger: logger,
	}

	return handler, nil
}

func (t TodoHandler) Handle(
	ctx context.Context,
	request *apiserver.Request,
) (*apiserver.Response, error) {
	todo := request.Body.(*Todo)
	todo.CreatedAt = time.Now()

	t.logger.Info("got todo with id %d", todo.Id)

	return apiserver.NewJsonResponse(todo), nil
}