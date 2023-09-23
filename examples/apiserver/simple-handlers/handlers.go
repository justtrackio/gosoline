package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/db-repo"
)

type JsonResponseFromMapHandler struct{}

func (h *JsonResponseFromMapHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, error error) {
	m := map[string]string{
		"status": "success",
	}

	return apiserver.NewJsonResponse(m), nil
}

type JsonResponseFromStructHandler struct{}

func (h *JsonResponseFromStructHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, error error) {
	obj := myTestStruct{
		Status: "success",
	}

	return apiserver.NewJsonResponse(obj), nil
}

type (
	JsonInputHandler struct{}
	inputEntity      struct {
		Message string `form:"message" binding:"required"`
	}
)

func (h *JsonInputHandler) GetInput() interface{} {
	return &inputEntity{}
}

func (h *JsonInputHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, error error) {
	input := request.Body.(*inputEntity)
	output := fmt.Sprintf("Thank you for submitting your message '%s', we will handle it with care!", input.Message)

	return apiserver.NewJsonResponse(map[string]string{
		"message": output,
	}), nil
}

type AdminAuthenticatedHandler struct{}

func (h *AdminAuthenticatedHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, error error) {
	m := map[string]bool{
		"authenticated": true,
	}

	return apiserver.NewJsonResponse(m), nil
}

type MyEntityHandler struct {
	repo db_repo.Repository[uint, *MyEntity]
}

type MyEntityCreateInput struct {
	Prop1 string `json:"prop1"`
	Prop2 string `json:"prop2"`
}

type MyEntityUpdateInput struct {
	Prop1 string `json:"prop1"`
}

func (h *MyEntityHandler) GetRepository() db_repo.Repository[uint, *MyEntity] {
	return h.repo
}

func (h *MyEntityHandler) TransformOutput(_ context.Context, model *MyEntity, apiView string) (output *MyEntity, err error) {
	return model, nil
}

func (h *MyEntityHandler) TransformCreate(_ context.Context, input *MyEntityCreateInput) (model *MyEntity, err error) {
	return &MyEntity{
		Prop1: input.Prop1,
		Prop2: input.Prop2,
	}, nil
}

func (h *MyEntityHandler) TransformUpdate(_ context.Context, input *MyEntityUpdateInput, model *MyEntity) (updated *MyEntity, err error) {
	model.Prop1 = input.Prop1

	return model, nil
}

func (h *MyEntityHandler) List(ctx context.Context, qb *db_repo.QueryBuilder, apiView string) (out []*MyEntity, err error) {
	return h.repo.Query(ctx, qb)
}

func (h *MyEntityHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, error error) {
	m := map[string]bool{
		"authenticated": true,
	}

	return apiserver.NewJsonResponse(m), nil
}
