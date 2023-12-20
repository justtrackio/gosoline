package main

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/apiserver/crud"
	"github.com/justtrackio/gosoline/pkg/db-repo"
)

type JsonResponseFromMapHandler struct{}

func (h *JsonResponseFromMapHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, err error) {
	m := map[string]string{
		"status": "success",
	}

	return apiserver.NewJsonResponse(m), nil
}

type JsonResponseFromStructHandler struct{}

func (h *JsonResponseFromStructHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, err error) {
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

func (h *JsonInputHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, err error) {
	input := request.Body.(*inputEntity)
	output := fmt.Sprintf("Thank you for submitting your message '%s', we will handle it with care!", input.Message)

	return apiserver.NewJsonResponse(map[string]string{
		"message": output,
	}), nil
}

type AdminAuthenticatedHandler struct{}

func (h *AdminAuthenticatedHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, err error) {
	m := map[string]bool{
		"authenticated": true,
	}

	return apiserver.NewJsonResponse(m), nil
}

type MyEntityHandler struct {
	repo crud.Repository
}

type MyEntityCreateInput struct {
	Prop1 string `json:"prop1"`
	Prop2 string `json:"prop2"`
}

type MyEntityUpdateInput struct {
	Prop1 string `json:"prop1"`
}

func (h *MyEntityHandler) GetRepository() crud.Repository {
	return h.repo
}

func (h *MyEntityHandler) GetModel() db_repo.ModelBased {
	return &MyEntity{}
}

func (h *MyEntityHandler) TransformOutput(_ context.Context, model db_repo.ModelBased, apiView string) (output interface{}, err error) {
	return model, nil
}

func (h *MyEntityHandler) GetCreateInput() interface{} {
	return &MyEntityCreateInput{}
}

func (h *MyEntityHandler) TransformCreate(_ context.Context, input interface{}, model db_repo.ModelBased) (err error) {
	i := input.(*MyEntityCreateInput)
	b := model.(*MyEntity)

	b.Prop1 = i.Prop1
	b.Prop2 = i.Prop2

	return
}

func (h *MyEntityHandler) GetUpdateInput() interface{} {
	return &MyEntityUpdateInput{}
}

func (h *MyEntityHandler) TransformUpdate(_ context.Context, input interface{}, model db_repo.ModelBased) (err error) {
	i := input.(*MyEntityUpdateInput)
	b := model.(*MyEntity)

	b.Prop1 = i.Prop1

	return
}

func (h *MyEntityHandler) List(ctx context.Context, qb *db_repo.QueryBuilder, apiView string) (out interface{}, err error) {
	res := make([]*MyEntity, 0)
	err = h.repo.Query(ctx, qb, &res)

	return res, err
}

func (h *MyEntityHandler) Handle(requestContext context.Context, request *apiserver.Request) (response *apiserver.Response, err error) {
	m := map[string]bool{
		"authenticated": true,
	}

	return apiserver.NewJsonResponse(m), nil
}
