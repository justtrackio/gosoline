package crud

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
	"net/http"
)

type createHandler struct {
	transformer Handler
}

func NewCreateHandler(transformer Handler, config cfg.Config, logger mon.Logger) gin.HandlerFunc {
	ch := createHandler{
		transformer: transformer,
	}

	return apiserver.CreateJsonHandler(ch, config, logger)
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

	exists := db.IsDuplicateEntryError(err)

	if exists {
		return apiserver.NewStatusResponse(http.StatusConflict), nil
	}

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
