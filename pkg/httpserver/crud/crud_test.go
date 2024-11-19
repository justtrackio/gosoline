package crud_test

import (
	"context"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type Model struct {
	db_repo.Model
	Name *string `json:"name"`
}

type Output struct {
	Id        *uint      `json:"id"`
	Name      *string    `json:"name"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
}

type CreateInput struct {
	Name *string `json:"name" binding:"required"`
}

type UpdateInput struct {
	Name *string `json:"name" binding:"required"`
}

type handler struct {
	Repo *mocks.Repository
}

func (h handler) GetRepository() crud.Repository {
	return h.Repo
}

func (h handler) GetModel() db_repo.ModelBased {
	return &Model{}
}

func (h handler) GetCreateInput() any {
	return &CreateInput{}
}

func (h handler) GetUpdateInput() any {
	return &UpdateInput{}
}

func (h handler) TransformCreate(_ context.Context, inp any, model db_repo.ModelBased) (err error) {
	input := inp.(*CreateInput)
	m := model.(*Model)

	m.Name = input.Name

	return nil
}

func (h handler) TransformUpdate(_ context.Context, inp any, model db_repo.ModelBased) (err error) {
	input := inp.(*UpdateInput)
	m := model.(*Model)

	m.Name = input.Name

	return nil
}

func (h handler) TransformOutput(_ context.Context, model db_repo.ModelBased, _ string) (any, error) {
	m := model.(*Model)

	out := &Output{
		Id:        m.Id,
		Name:      m.Name,
		UpdatedAt: m.UpdatedAt,
		CreatedAt: m.CreatedAt,
	}

	return out, nil
}

func (h handler) List(_ context.Context, _ *db_repo.QueryBuilder, _ string) (any, error) {
	date, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	if err != nil {
		panic(err)
	}

	return []Model{
		{
			Model: db_repo.Model{
				Id: mdl.Box(uint(1)),
				Timestamps: db_repo.Timestamps{
					UpdatedAt: mdl.Box(date),
					CreatedAt: mdl.Box(date),
				},
			},
			Name: mdl.Box("foobar"),
		},
	}, nil
}

func newHandler(t *testing.T) handler {
	repo := mocks.NewRepository(t)

	return handler{
		Repo: repo,
	}
}
