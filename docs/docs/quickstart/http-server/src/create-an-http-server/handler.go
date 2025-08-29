// snippet-start: imports
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

// snippet-end: imports

// snippet-start: todo struct
type Todo struct {
	Id        int    `form:"id"`
	Text      string `form:"text"`
	CreatedAt time.Time
}

// snippet-end: todo struct

// snippet-start: todo handler
type TodoHandler struct {
	logger log.Logger
}

// snippet-end: todo handler

// snippet-start: get input
func (t TodoHandler) GetInput() any {
	return &Todo{}
}

// snippet-end: get input

// snippet-start: new todo handler
func NewTodoHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*TodoHandler, error) {
	handler := &TodoHandler{
		logger: logger,
	}

	return handler, nil
}

// snippet-end: new todo handler

// snippet-start: handle
func (t TodoHandler) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	// Initialize a Todo struct from the request body
	todo := request.Body.(*Todo)

	// Set its CreatedAt to now
	todo.CreatedAt = time.Now()

	// The error gets transformed within the http server to an HTTP 500 response
	if todo.Id <= 0 {
		return nil, fmt.Errorf("invalid id")
	}

	// Log the request using the TodoHandler struct's logger
	t.logger.Info(ctx, "got todo with id %d", todo.Id)

	// Return a Json response object with information from the Todo struct
	return httpserver.NewJsonResponse(todo), nil
}

// snippet-end: handle
