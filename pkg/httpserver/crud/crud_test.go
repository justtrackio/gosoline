package crud_test

import (
	"context"
	"testing"
	"time"

	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	dbRepoMocks "github.com/justtrackio/gosoline/pkg/db-repo/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type Model struct {
	dbRepo.Model
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
	Repo *dbRepoMocks.Repository[uint, *Model]
}

func (h handler) GetRepository() dbRepo.Repository[uint, *Model] {
	return h.Repo
}

func (h handler) TransformCreate(_ context.Context, inp *CreateInput) (*Model, error) {
	return &Model{
		Name: inp.Name,
	}, nil
}

func (h handler) TransformUpdate(_ context.Context, inp *UpdateInput, model *Model) (*Model, error) {
	model.Name = inp.Name

	return model, nil
}

func (h handler) TransformOutput(_ context.Context, model *Model, _ string) (output Output, err error) {
	out := Output{
		Id:        model.Id,
		Name:      model.Name,
		UpdatedAt: model.UpdatedAt,
		CreatedAt: model.CreatedAt,
	}

	return out, nil
}

func (h handler) List(_ context.Context, _ *dbRepo.QueryBuilder, _ string) (out []Output, err error) {
	date, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
	if err != nil {
		panic(err)
	}

	return []Output{
		{
			Id:        mdl.Box(uint(1)),
			Name:      mdl.Box("foobar"),
			UpdatedAt: mdl.Box(date),
			CreatedAt: mdl.Box(date),
		},
	}, nil
}

func newHandler(t *testing.T) handler {
	repo := dbRepoMocks.NewRepository[uint, *Model](t)

	return handler{
		Repo: repo,
	}
}
