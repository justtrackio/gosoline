package main

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/apiserver/crud"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

var settings = db_repo.Settings{
	Metadata: db_repo.Metadata{
		ModelId: mdl.ModelId{
			Name: "todo",
		},
		TableName:  "todos",
		PrimaryKey: "todos.id",
	},
}

type Todo struct {
	db_repo.Model
	Text    string
	DueDate time.Time
}

type CreateInput struct {
	Text    string    `form:"text"`
	DueDate time.Time `form:"dueDate"`
}

type UpdateInput struct {
	Text string `form:"text"`
}

type TodoCrudHandlerV0 struct {
	repo db_repo.Repository
}

func NewTodoCrudHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*TodoCrudHandlerV0, error) {
	var err error
	var repo db_repo.Repository

	if repo, err = db_repo.New(config, logger, settings); err != nil {
		return nil, fmt.Errorf("can not create db_repo.Repositorys: %w", err)
	}

	handler := &TodoCrudHandlerV0{
		repo: repo,
	}

	return handler, nil
}

func (h TodoCrudHandlerV0) GetRepository() crud.Repository {
	return h.repo
}

func (h TodoCrudHandlerV0) GetModel() db_repo.ModelBased {
	return &Todo{}
}

func (h TodoCrudHandlerV0) GetCreateInput() interface{} {
	return &CreateInput{}
}

func (h TodoCrudHandlerV0) TransformCreate(ctx context.Context, input interface{}, model db_repo.ModelBased) error {
	in := input.(*CreateInput)
	m := model.(*Todo)

	m.Text = in.Text
	m.DueDate = in.DueDate

	return nil
}

func (h TodoCrudHandlerV0) GetUpdateInput() interface{} {
	return &UpdateInput{}
}

func (h TodoCrudHandlerV0) TransformUpdate(ctx context.Context, input interface{}, model db_repo.ModelBased) error {
	in := input.(*UpdateInput)
	m := model.(*Todo)

	m.Text = in.Text

	return nil
}

func (h TodoCrudHandlerV0) TransformOutput(ctx context.Context, model db_repo.ModelBased, apiView string) (interface{}, error) {
	return model, nil
}

func (h TodoCrudHandlerV0) List(ctx context.Context, qb *db_repo.QueryBuilder, apiView string) (interface{}, error) {
	var err error
	result := make([]*Todo, 0)

	if err = h.repo.Query(ctx, qb, &result); err != nil {
		return nil, fmt.Errorf("can not query todo items: %w", err)
	}

	out := make([]interface{}, len(result))
	for i, res := range result {
		if out[i], err = h.TransformOutput(ctx, res, apiView); err != nil {
			return nil, err
		}
	}

	return out, nil
}