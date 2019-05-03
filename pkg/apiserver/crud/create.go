package crud

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/gin-gonic/gin"
)

type createHandler struct {
	transformer Handler
}

func NewCreateHandler(transformer Handler) gin.HandlerFunc {
	ch := createHandler{
		transformer: transformer,
	}

	return apiserver.CreateJsonHandler(ch)
}

func (ch createHandler) GetInput() interface{} {
	return ch.transformer.GetCreateInput()
}

func (ch createHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	model := ch.transformer.GetModel()
	err := ch.transformer.TransformCreate(request.Body, model)

	if err != nil {
		return nil, err
	}

	repo := ch.transformer.GetRepository()
	err = repo.Create(ctx, model)

	if err != nil {
		return nil, err
	}

	reload := ch.transformer.GetModel()
	err = repo.Read(ctx, model.GetId(), reload)

	if err != nil {
		return nil, err
	}

	apiView := getApiViewFromHeader(request.Header)
	out, err := ch.transformer.TransformOutput(reload, apiView)

	if err != nil {
		return nil, err
	}

	return apiserver.NewJsonResponse(out), nil
}
