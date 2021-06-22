package crud_test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/apiserver/crud"
	"github.com/applike/gosoline/pkg/apiserver/crud/mocks"
	"github.com/applike/gosoline/pkg/db-repo"
	logMocks "github.com/applike/gosoline/pkg/log/mocks"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"net/http"
	"testing"
	"time"
)

type Model struct {
	db_repo.Model
	Name *string `json:"name"`
}

type Output struct {
	Id        *uint      `json:"id"`
	Name      *string    `json:"name"`
	UpdatedAt *time.Time `json:"updatedAt"`
	CreatedAt *time.Time `json:"createdAt"`
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

func (h Handler) TransformCreate(inp interface{}, model db_repo.ModelBased) (err error) {
	input := inp.(*CreateInput)
	m := model.(*Model)

	m.Name = input.Name

	return nil
}

func (h Handler) TransformUpdate(inp interface{}, model db_repo.ModelBased) (err error) {
	input := inp.(*UpdateInput)
	m := model.(*Model)

	m.Name = input.Name

	return nil
}

func (h Handler) TransformOutput(model db_repo.ModelBased, _ string) (interface{}, error) {
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
				Id: mdl.Uint(1),
				Timestamps: db_repo.Timestamps{
					UpdatedAt: mdl.Time(date),
					CreatedAt: mdl.Time(date),
				},
			},
			Name: mdl.String("foobar"),
		},
	}, nil
}

func NewTransformer() Handler {
	repo := new(mocks.Repository)

	return Handler{
		Repo: repo,
	}
}

var id1 = mdl.Uint(1)

func TestCreateHandler_Handle(t *testing.T) {
	model := &Model{
		Name: mdl.String("foobar"),
	}

	logger := logMocks.NewLoggerMockedAll()
	transformer := NewTransformer()

	transformer.Repo.On("Create", mock.AnythingOfType("*context.emptyCtx"), model).Run(func(args mock.Arguments) {
		model := args.Get(1).(*Model)
		model.Id = mdl.Uint(1)
	}).Return(nil)
	transformer.Repo.On("Read", mock.AnythingOfType("*context.emptyCtx"), mdl.Uint(1), &Model{}).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = mdl.Uint(1)
		model.Name = mdl.String("foobar")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	handler := crud.NewCreateHandler(logger, transformer)

	body := `{"name": "foobar"}`
	response := apiserver.HttpTest("POST", "/create", "/create", body, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestCreateHandler_Handle_ValidationError(t *testing.T) {
	model := &Model{
		Name: mdl.String("foobar"),
	}

	logger := logMocks.NewLoggerMockedAll()
	transformer := NewTransformer()

	transformer.Repo.On("Create", mock.AnythingOfType("*context.emptyCtx"), model).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})

	handler := crud.NewCreateHandler(logger, transformer)

	body := `{"name": "foobar"}`
	response := apiserver.HttpTest("POST", "/create", "/create", body, handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"err":"validation: invalid foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestReadHandler_Handle(t *testing.T) {
	model := &Model{}

	logger := logMocks.NewLoggerMockedAll()
	transformer := NewTransformer()
	transformer.Repo.On("Read", mock.AnythingOfType("*context.emptyCtx"), mdl.Uint(1), model).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = mdl.Uint(1)
		model.Name = mdl.String("foobar")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	handler := crud.NewReadHandler(logger, transformer)

	response := apiserver.HttpTest("GET", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestUpdateHandler_Handle(t *testing.T) {
	readModel := &Model{}
	updateModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Uint(1),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.String("updated"),
	}

	logger := logMocks.NewLoggerMockedAll()
	transformer := NewTransformer()

	transformer.Repo.On("Update", mock.AnythingOfType("*context.emptyCtx"), updateModel).Return(nil)
	transformer.Repo.On("Read", mock.AnythingOfType("*context.emptyCtx"), mdl.Uint(1), readModel).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = mdl.Uint(1)
		model.Name = mdl.String("updated")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	handler := crud.NewUpdateHandler(logger, transformer)

	body := `{"name": "updated"}`
	response := apiserver.HttpTest("PUT", "/:id", "/1", body, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id":1,"updatedAt":"0001-01-01T00:00:00Z","createdAt":"0001-01-01T00:00:00Z","name":"updated"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestUpdateHandler_Handle_ValidationError(t *testing.T) {
	readModel := &Model{}
	updateModel := &Model{
		Model: db_repo.Model{
			Id: mdl.Uint(1),
			Timestamps: db_repo.Timestamps{
				UpdatedAt: &time.Time{},
				CreatedAt: &time.Time{},
			},
		},
		Name: mdl.String("updated"),
	}

	logger := logMocks.NewLoggerMockedAll()
	transformer := NewTransformer()

	transformer.Repo.On("Update", mock.AnythingOfType("*context.emptyCtx"), updateModel).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})
	transformer.Repo.On("Read", mock.AnythingOfType("*context.emptyCtx"), mdl.Uint(1), readModel).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = mdl.Uint(1)
		model.Name = mdl.String("updated")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)

	handler := crud.NewUpdateHandler(logger, transformer)

	body := `{"name": "updated"}`
	response := apiserver.HttpTest("PUT", "/:id", "/1", body, handler)

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
		Name: mdl.String("foobar"),
	}

	logger := logMocks.NewLoggerMockedAll()
	transformer := NewTransformer()
	transformer.Repo.On("Read", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*uint"), model).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = id1
		model.Name = mdl.String("foobar")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)
	transformer.Repo.On("Delete", mock.AnythingOfType("*context.emptyCtx"), deleteModel).Return(nil)

	handler := crud.NewDeleteHandler(logger, transformer)

	response := apiserver.HttpTest("DELETE", "/:id", "/1", "", handler)

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
		Name: mdl.String("foobar"),
	}

	logger := logMocks.NewLoggerMockedAll()
	transformer := NewTransformer()
	transformer.Repo.On("Read", mock.AnythingOfType("*context.emptyCtx"), mock.AnythingOfType("*uint"), model).Run(func(args mock.Arguments) {
		model := args.Get(2).(*Model)
		model.Id = id1
		model.Name = mdl.String("foobar")
		model.UpdatedAt = &time.Time{}
		model.CreatedAt = &time.Time{}
	}).Return(nil)
	transformer.Repo.On("Delete", mock.AnythingOfType("*context.emptyCtx"), deleteModel).Return(&validation.Error{
		Errors: []error{fmt.Errorf("invalid foobar")},
	})

	handler := crud.NewDeleteHandler(logger, transformer)

	response := apiserver.HttpTest("DELETE", "/:id", "/1", "", handler)

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"err":"validation: invalid foobar"}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}

func TestListHandler_Handle(t *testing.T) {
	logger := logMocks.NewLoggerMockedAll()
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
	transformer.Repo.On("Count", mock.AnythingOfType("*context.emptyCtx"), qb, &Model{}).Return(1, nil)

	body := `{"filter":{"matches":[{"values":["foobar"],"dimension":"name","operator":"="}],"bool":"and"},"order":[{"field":"name","direction":"ASC"}],"page":{"offset":0,"limit":2}}`
	response := apiserver.HttpTest("PUT", "/:id", "/1", body, handler)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"total":1,"results":[{"Id":1,"UpdatedAt":"2006-01-02T15:04:05Z","CreatedAt":"2006-01-02T15:04:05Z","name":"foobar"}]}`, response.Body.String())

	transformer.Repo.AssertExpectations(t)
}
