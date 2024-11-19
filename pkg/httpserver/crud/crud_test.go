package crud_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	dbRepoMocks "github.com/justtrackio/gosoline/pkg/db-repo/mocks"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/validation"
	"github.com/stretchr/testify/assert"
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

type Handler struct {
	Repo *dbRepoMocks.Repository[uint, *Model]
}

func (h Handler) GetRepository() dbRepo.Repository[uint, *Model] {
	return h.Repo
}

func (h Handler) TransformCreate(_ context.Context, inp *CreateInput) (*Model, error) {
	return &Model{
		Name: inp.Name,
	}, nil
}

func (h Handler) TransformUpdate(_ context.Context, inp *UpdateInput, model *Model) (*Model, error) {
	model.Name = inp.Name

	return model, nil
}

func (h Handler) TransformOutput(_ context.Context, model *Model, _ string) (output Output, err error) {
	out := Output{
		Id:        model.Id,
		Name:      model.Name,
		UpdatedAt: model.UpdatedAt,
		CreatedAt: model.CreatedAt,
	}

	return out, nil
}

func (h Handler) List(_ context.Context, _ *dbRepo.QueryBuilder, _ string) (out []Output, err error) {
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

func NewTransformer() Handler {
	repo := new(dbRepoMocks.Repository[uint, *Model])

	return Handler{
		Repo: repo,
	}
}

func TestCreateHandler_Handle(t *testing.T) {
	model := &Model{
		Name: mdl.Box("foobar"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()

	transformer.Repo.EXPECT().Create(context.Background(), model).Run(func(ctx context.Context, value *Model) {
		value.Id = mdl.Box(uint(1))
	}).Return(nil)
	transformer.Repo.EXPECT().Read(context.Background(), uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}, nil)

	handler := crud.NewCreateHandler[CreateInput, Output, uint, *Model](logger, transformer)

	body := `{"name": "foobar"}`
	response := httpserver.HttpTest("POST", "/create", "/create", body, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestCreateHandler_Handle_ValidationError(t *testing.T) {
	model := &Model{
		Name: mdl.Box("foobar"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()

	transformer.Repo.EXPECT().Create(context.Background(), model).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})

	handler := crud.NewCreateHandler[CreateInput, Output, uint, *Model](logger, transformer)

	body := `{"name": "foobar"}`
	response := httpserver.HttpTest("POST", "/create", "/create", body, handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"err":"validation: invalid foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestReadHandler_Handle(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()
	transformer.Repo.EXPECT().Read(context.Background(), uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}, nil)

	handler := crud.NewReadHandler[Output, uint, *Model](logger, transformer)

	response := httpserver.HttpTest("GET", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestUpdateHandler_Handle(t *testing.T) {
	updateModel := &Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()

	transformer.Repo.EXPECT().Update(context.Background(), updateModel).Return(nil)
	transformer.Repo.EXPECT().Read(context.Background(), uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}, nil)

	handler := crud.NewUpdateHandler[UpdateInput, Output, uint, *Model](logger, transformer)

	body := `{"name": "updated"}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"updated"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestUpdateHandler_Handle_ValidationError(t *testing.T) {
	updateModel := &Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()

	transformer.Repo.EXPECT().Update(context.Background(), updateModel).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})
	transformer.Repo.EXPECT().Read(context.Background(), uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}, nil)

	handler := crud.NewUpdateHandler[UpdateInput, Output, uint, *Model](logger, transformer)

	body := `{"name": "updated"}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"err":"validation: invalid foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestDeleteHandler_Handle(t *testing.T) {
	deleteModel := &Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()

	transformer.Repo.EXPECT().Read(context.Background(), uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}, nil)
	transformer.Repo.EXPECT().Delete(context.Background(), deleteModel).Return(nil)

	handler := crud.NewDeleteHandler[Output, uint, *Model](logger, transformer)

	response := httpserver.HttpTest("DELETE", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestDeleteHandler_Handle_ValidationError(t *testing.T) {
	deleteModel := &Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()

	transformer.Repo.EXPECT().Read(context.Background(), uint(1)).Return(&Model{
		Model: dbRepo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: dbRepo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}, nil)
	transformer.Repo.EXPECT().Delete(context.Background(), deleteModel).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})

	handler := crud.NewDeleteHandler[Output, uint, *Model](logger, transformer)

	response := httpserver.HttpTest("DELETE", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"err":"validation: invalid foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestListHandler_Handle(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()
	handler := crud.NewListHandler[Output, uint, *Model](logger, transformer)

	qb := dbRepo.NewQueryBuilder()
	qb.Table("footable")
	qb.Where("(((name = ?)))", "foobar")
	qb.GroupBy("id")
	qb.OrderBy("name", "ASC")
	qb.Page(0, 2)

	transformer.Repo.EXPECT().GetMetadata().Return(dbRepo.Metadata{
		TableName:  "footable",
		PrimaryKey: "id",
		Mappings: dbRepo.FieldMappings{
			"id":   dbRepo.NewFieldMapping("id"),
			"name": dbRepo.NewFieldMapping("name"),
		},
	})

	transformer.Repo.EXPECT().Count(context.Background(), qb).Return(1, nil)

	body := `{"filter":{"matches":[{"values":["foobar"],"dimension":"name","operator":"="}],"bool":"and"},"order":[{"field":"name","direction":"ASC"}],"page":{"offset":0,"limit":2}}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"total":1,"results":[{"id":1,"updatedAt":"2006-01-02T15:04:05Z","createdAt":"2006-01-02T15:04:05Z","name":"foobar"}]}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}
