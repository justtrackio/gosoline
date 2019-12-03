package crud

import (
	"context"
	"errors"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/gin-gonic/gin"
)

type deleteHandler struct {
	transformer BaseHandler
}

func NewDeleteHandler(transformer BaseHandler) gin.HandlerFunc {
	dh := deleteHandler{
		transformer: transformer,
	}

	return apiserver.CreateHandler(dh)
}

func (dh deleteHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	id, valid := apiserver.GetUintFromRequest(request, "id")

	if !valid {
		return nil, errors.New("no valid id provided")
	}

	repo := dh.transformer.GetRepository()
	model := dh.transformer.GetModel()

	err := repo.Read(ctx, id, model)

	if err != nil {
		return nil, err
	}

	err = repo.Delete(ctx, model)

	if err != nil {
		return nil, err
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := dh.transformer.TransformOutput(model, apiView)

	if err != nil {
		return nil, err
	}

	return apiserver.NewJsonResponse(out), nil
}
