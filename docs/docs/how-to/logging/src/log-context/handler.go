package main

import (
	"context"
	"fmt"
	"time"

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

// snippet-start: crud handler v0
type TodoCrudHandlerV0 struct {
	// highlight-next-line
	logger log.Logger
	repo   db_repo.Repository[uint, *Todo]
}

// snippet-end: crud handler v0

// snippet-start: new todo crud handler
func NewTodoCrudHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*TodoCrudHandlerV0, error) {
	var err error
	var repo db_repo.Repository[uint, *Todo]

	if repo, err = db_repo.New[uint, *Todo](ctx, config, logger, settings); err != nil {
		return nil, fmt.Errorf("can not create db_repo.Repositorys: %w", err)
	}

	handler := &TodoCrudHandlerV0{
		// highlight-next-line
		logger: logger,
		repo:   repo,
	}

	return handler, nil
}

// snippet-end: new todo crud handler

func (h TodoCrudHandlerV0) GetRepository() db_repo.Repository[uint, *Todo] {
	return h.repo
}

// snippet-start: truncate
func truncate(ctx context.Context, text string) string {
	r := []rune(text)
	length := len(r)

	log.MutateContextFields(ctx, map[string]any{
		"original_length": length,
	})

	if length > 50 {
		text = string(r[:50]) + "..."
	}

	return text
}

// snippet-end: truncate

// snippet-start: transform create
func (h TodoCrudHandlerV0) TransformCreate(ctx context.Context, in *CreateInput) (*Todo, error) {
	// highlight-start
	localCtx := log.InitContext(ctx)
	m := &Todo{
		Text: truncate(localCtx, in.Text),
		// highlight-end
		DueDate: in.DueDate,
	}

	return m, nil
}

// snippet-end: transform create

// snippet-start: transform update
func (h TodoCrudHandlerV0) TransformUpdate(ctx context.Context, in *UpdateInput, m *Todo) (*Todo, error) {
	// highlight-start
	localCtx := log.InitContext(ctx)
	m.Text = truncate(localCtx, in.Text)
	// highlight-end

	return m, nil
}

// snippet-end: transform update

func (h TodoCrudHandlerV0) TransformOutput(ctx context.Context, model *Todo, apiView string) (*Todo, error) {
	return model, nil
}
func (h TodoCrudHandlerV0) List(ctx context.Context, qb *db_repo.QueryBuilder, apiView string) ([]*Todo, error) {
	var err error
	var result []*Todo

	// Query the database using a Context and a QueryBuilder. If it finds the results, it sets them on result. Otherwise, it returns an error.
	if result, err = h.repo.Query(ctx, qb); err != nil {
		return nil, fmt.Errorf("can not query todo items: %w", err)
	}

	// Transform each result with TransformOutput().
	out := make([]*Todo, len(result))
	for i, res := range result {
		if out[i], err = h.TransformOutput(ctx, res, apiView); err != nil {
			return nil, err
		}
	}

	// Return the transformed results.
	return out, nil
}
