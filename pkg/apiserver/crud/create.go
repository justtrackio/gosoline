package crud

import (
	"context"
	"errors"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/db"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/validation"
	"github.com/gin-gonic/gin"
	"net/http"
)

type createHandler struct {
	transformer CreateHandler
	logger      log.Logger
}

func NewCreateHandler(logger log.Logger, transformer CreateHandler) gin.HandlerFunc {
	ch := createHandler{
		transformer: transformer,
		logger:      logger,
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

	if db.IsDuplicateEntryError(err) {
		return apiserver.NewStatusResponse(http.StatusConflict), nil
	}

	if errors.Is(err, &validation.Error{}) {
		return apiserver.GetErrorHandler()(http.StatusBadRequest, err), nil
	}

	if err != nil {
		return nil, err
	}

	reload := ch.transformer.GetModel()
	err = repo.Read(ctx, model.GetId(), reload)

	if err != nil {
		return nil, err
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := ch.transformer.TransformOutput(reload, apiView)

	if err != nil {
		return nil, err
	}

	return apiserver.NewJsonResponse(out), nil
}
