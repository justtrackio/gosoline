package crud

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var ErrNotFound = errors.New("model not found")

type readHandler struct {
	transformer Handler
}

func NewReadHandler(transformer Handler) gin.HandlerFunc {
	rh := readHandler{
		transformer: transformer,
	}

	return apiserver.CreateHandler(rh)
}

func (rh readHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	id, valid := apiserver.GetUintFromRequest(request, "id")

	if !valid {
		return nil, errors.New("no valid id provided")
	}

	repo := rh.transformer.GetRepository()
	model := rh.transformer.GetModel()
	err := repo.Read(ctx, id, model)

	if err != nil {
		return nil, err
	}

	apiView := getApiViewFromHeader(request.Header)
	out, err := rh.transformer.TransformOutput(model, apiView)

	if err != nil {
		return nil, err
	}

	return apiserver.NewJsonResponse(out), nil
}
