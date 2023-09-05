package main

import (
	"context"
	"time"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

// the struct used to bind the input: ?id=5&text=todo
type Todo struct {
	Id        int    `form:"id"` // form tag defines the query param name
	Text      string `form:"text"`
	CreatedAt time.Time
}

// the actual todohandler struct
type TodoHandler struct {
	logger log.Logger
}

// returns the input instance to use
func (t TodoHandler) GetInput() interface{} {
	return &Todo{}
}

func NewTodoHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*TodoHandler, error) {
	handler := &TodoHandler{
		logger: logger,
	}

	return handler, nil
}

// this one is called for every request
func (t TodoHandler) Handle(
	// context of this request
	ctx context.Context,
	// request information (body, params, headers, cookies, client ip, ...)
	request *apiserver.Request,
) (*apiserver.Response, error) {
	// get the input data from request body
	todo := request.Body.(*Todo)
	todo.CreatedAt = time.Now()

	t.logger.Info("got todo with id %d", todo.Id)

	// write a json response with the todo
	return apiserver.NewJsonResponse(todo), nil
}
