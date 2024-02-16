// snippet-start: imports
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	db_repo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

// snippet-end: imports

// snippet-start: settings
var settings = db_repo.Settings{
	Metadata: db_repo.Metadata{
		ModelId: mdl.ModelId{
			Name: "todo",
		},
		TableName:  "todos",
		PrimaryKey: "todos.id",
	},
}

// snippet-end: settings

// snippet-start: todo
type Todo struct {
	db_repo.Model
	Text    string
	DueDate time.Time
}

// snippet-end: todo

// snippet-start: create and update
type CreateInput struct {
	Text    string    `form:"text"`
	DueDate time.Time `form:"dueDate"`
}

type UpdateInput struct {
	Text string `form:"text"`
}

// snippet-end: create and update

// snippet-start: crud handler
type TodoCrudHandlerV0 struct {
	repo db_repo.Repository
}

// snippet-end: crud handler

// snippet-start: todo constructor
func NewTodoCrudHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*TodoCrudHandlerV0, error) {
	// Declare `error` and `repo` variables.
	var err error
	var repo db_repo.Repository

	// Try to create a new `Repository` given a configuration, a logger, and settings. If there is an error, you return it.
	if repo, err = db_repo.New(config, logger, settings); err != nil {
		return nil, fmt.Errorf("can not create db_repo.Repositorys: %w", err)
	}

	// Create a new `TodoCrudHandlerV0` with that repo.
	handler := &TodoCrudHandlerV0{
		repo: repo,
	}

	// Return the handler.
	return handler, nil
}

// snippet-end: todo constructor

// snippet-start: get repo
func (h TodoCrudHandlerV0) GetRepository() crud.Repository {
	return h.repo
}

// snippet-end: get repo

// snippet-start: get model
func (h TodoCrudHandlerV0) GetModel() db_repo.ModelBased {
	return &Todo{}
}

// snippet-end: get model

// snippet-start: create input
func (h TodoCrudHandlerV0) GetCreateInput() interface{} {
	return &CreateInput{}
}

// snippet-end: create input

// snippet-start: transform create
func (h TodoCrudHandlerV0) TransformCreate(ctx context.Context, input interface{}, model db_repo.ModelBased) error {
	in := input.(*CreateInput)
	m := model.(*Todo)

	m.Text = in.Text
	m.DueDate = in.DueDate

	return nil
}

// snippet-end: transform create

// snippet-start: get update input
func (h TodoCrudHandlerV0) GetUpdateInput() interface{} {
	return &UpdateInput{}
}

// snippet-end: get update input

// snippet-start: transform update
func (h TodoCrudHandlerV0) TransformUpdate(ctx context.Context, input interface{}, model db_repo.ModelBased) error {
	in := input.(*UpdateInput)
	m := model.(*Todo)

	m.Text = in.Text

	return nil
}

// snippet-end: transform update

// snippet-start: transform output
func (h TodoCrudHandlerV0) TransformOutput(ctx context.Context, model db_repo.ModelBased, apiView string) (interface{}, error) {
	return model, nil
}

// snippet-end: transform output

// snippet-start: list
func (h TodoCrudHandlerV0) List(ctx context.Context, qb *db_repo.QueryBuilder, apiView string) (interface{}, error) {
	var err error

	// Instatiate a list of Todo objects, called result.
	result := make([]*Todo, 0)

	// Query the database using a Context and a QueryBuilder. If it finds the results, it sets them on result. Otherwise, it returns an error.
	if err = h.repo.Query(ctx, qb, &result); err != nil {
		return nil, fmt.Errorf("can not query todo items: %w", err)
	}

	// Transform each result with TransformOutput().
	out := make([]interface{}, len(result))
	for i, res := range result {
		if out[i], err = h.TransformOutput(ctx, res, apiView); err != nil {
			return nil, err
		}
	}

	// Return the transformed results.
	return out, nil
}

// snippet-end: list
