package crud_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

type Handler struct {
	Repo *mocks.Repository
}

func (h Handler) GetRepository() crud.Repository {
	return h.Repo
}

func (h Handler) GetModel() db_repo.ModelBased {
	return &Model{}
}

func (h Handler) GetCreateInput() interface{} {
	return &CreateInput{}
}

func (h Handler) GetUpdateInput() interface{} {
	return &UpdateInput{}
}

func (h Handler) TransformCreate(_ctx context.Context, inp interface{}, model db_repo.ModelBased) (err error) {
	input := inp.(*CreateInput)
	m := model.(*Model)

	m.Name = input.Name

	return nil
}

func (h Handler) TransformUpdate(_ context.Context, inp interface{}, model db_repo.ModelBased) (err error) {
	input := inp.(*UpdateInput)
	m := model.(*Model)

	m.Name = input.Name

	return nil
}

func (h Handler) TransformOutput(ctx context.Context, model db_repo.ModelBased, _ string) (interface{}, error) {
	m := model.(*Model)

	out := &Output{
		Id:        m.Id,
		Name:      m.Name,
		UpdatedAt: m.UpdatedAt,
		CreatedAt: m.CreatedAt,
	}

	return out, nil
}

func (h Handler) List(_ context.Context, _ *db_repo.QueryBuilder, _ string) (interface{}, error) {
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

func NewTransformer() Handler {
	repo := new(mocks.Repository)

	return Handler{
		Repo: repo,
	}
}

var id1 = mdl.Box(uint(1))

func TestCreateHandler_Handle(t *testing.T) {
	model := &Model{
		Name: mdl.Box("foobar"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()

	transformer.Repo.On("Create", mock.AnythingOfType("context.backgroundCtx"), model).Run(func(args mock.Arguments) {
		model := args.Get(1).(*Model)
		model.Id = mdl.Box(uint(1))
	}).Return(nil)
	transformer.Repo.On("Read", mock.AnythingOfType("context.backgroundCtx"), mdl.Box(uint(1)), &Model{}).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = mdl.Box(uint(1))
		model.Name = mdl.Box("foobar")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	handler := crud.NewCreateHandler(logger, transformer)

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

	transformer.Repo.On("Create", mock.AnythingOfType("context.backgroundCtx"), model).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})

	handler := crud.NewCreateHandler(logger, transformer)

	body := `{"name": "foobar"}`
	response := httpserver.HttpTest("POST", "/create", "/create", body, handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"err":"validation: invalid foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestReadHandler_Handle(t *testing.T) {
	model := &Model{}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()
	transformer.Repo.On("Read", mock.AnythingOfType("context.backgroundCtx"), mdl.Box(uint(1)), model).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = mdl.Box(uint(1))
		model.Name = mdl.Box("foobar")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	handler := crud.NewReadHandler(logger, transformer)

	response := httpserver.HttpTest("GET", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestUpdateHandler_Handle(t *testing.T) {
	readModel := &Model{}
	updateModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()

	transformer.Repo.On("Update", mock.AnythingOfType("context.backgroundCtx"), updateModel).Return(nil)
	transformer.Repo.On("Read", mock.AnythingOfType("context.backgroundCtx"), mdl.Box(uint(1)), readModel).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = mdl.Box(uint(1))
		model.Name = mdl.Box("updated")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	handler := crud.NewUpdateHandler(logger, transformer)

	body := `{"name": "updated"}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"updated"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestUpdateHandler_Handle_ValidationError(t *testing.T) {
	readModel := &Model{}
	updateModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Box(uint(1)),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("updated"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()

	transformer.Repo.On("Update", mock.AnythingOfType("context.backgroundCtx"), updateModel).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})
	transformer.Repo.On("Read", mock.AnythingOfType("context.backgroundCtx"), mdl.Box(uint(1)), readModel).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = mdl.Box(uint(1))
		model.Name = mdl.Box("updated")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	handler := crud.NewUpdateHandler(logger, transformer)

	body := `{"name": "updated"}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"err":"validation: invalid foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestDeleteHandler_Handle(t *testing.T) {
	model := &Model{}
	deleteModel := &Model{
		Model: db_repo.Model{
			Id: id1,
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()
	transformer.Repo.On("Read", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("*uint"), model).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = id1
		model.Name = mdl.Box("foobar")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)
	transformer.Repo.On("Delete", mock.AnythingOfType("context.backgroundCtx"), deleteModel).Return(nil)

	handler := crud.NewDeleteHandler(logger, transformer)

	response := httpserver.HttpTest("DELETE", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestDeleteHandler_Handle_ValidationError(t *testing.T) {
	model := &Model{}
	deleteModel := &Model{
		Model: db_repo.Model{
			Id: id1,
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.Box("foobar"),
	}

	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()
	transformer.Repo.On("Read", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("*uint"), model).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = id1
		model.Name = mdl.Box("foobar")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)
	transformer.Repo.On("Delete", mock.AnythingOfType("context.backgroundCtx"), deleteModel).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})

	handler := crud.NewDeleteHandler(logger, transformer)

	response := httpserver.HttpTest("DELETE", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"err":"validation: invalid foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestListHandler_Handle(t *testing.T) {
	logger := logMocks.NewLoggerMock(logMocks.WithMockAll, logMocks.WithTestingT(t))
	transformer := NewTransformer()
	handler := crud.NewListHandler(logger, transformer)

	qb := db_repo.NewQueryBuilder()
	qb.Table("footable")
	qb.Where("(((name = ?)))", "foobar")
	qb.GroupBy("id")
	qb.OrderBy("name", "ASC")
	qb.Page(0, 2)

	transformer.Repo.On("GetMetadata").Return(db_repo.Metadata{
		TableName:  "footable",
		PrimaryKey: "id",
		Mappings: db_repo.FieldMappings{
			"id":   db_repo.NewFieldMapping("id"),
			"name": db_repo.NewFieldMapping("name"),
		},
	})
	transformer.Repo.On("Count", mock.AnythingOfType("context.backgroundCtx"), qb, &Model{}).Return(1, nil)

	body := `{"filter":{"matches":[{"values":["foobar"],"dimension":"name","operator":"="}],"bool":"and"},"order":[{"field":"name","direction":"ASC"}],"page":{"offset":0,"limit":2}}`
	response := httpserver.HttpTest("PUT", "/:id", "/1", body, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"total":1,"results":[{"Id":1,"UpdatedAt":"2006-01-02T15:04:05Z","CreatedAt":"2006-01-02T15:04:05Z","name":"foobar"}]}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}
