package crud

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type createHandler struct {
	logger      log.Logger
	transformer CreateHandler
}

func NewCreateHandler(logger log.Logger, transformer CreateHandler) gin.HandlerFunc {
	ch := createHandler{
		transformer: transformer,
		logger:      logger,
	}

	return httpserver.CreateJsonHandler(ch)
}

func (ch createHandler) GetInput() interface{} {
	return ch.transformer.GetCreateInput()
}

func (ch createHandler) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	model := ch.transformer.GetModel()
	err := ch.transformer.TransformCreate(ctx, request.Body, model)
	if err != nil {
		return nil, err
	}

	repo := ch.transformer.GetRepository()
	err = repo.Create(ctx, model)

	if db.IsDuplicateEntryError(err) {
		return httpserver.NewStatusResponse(http.StatusConflict), nil
	}

	if errors.Is(err, &validation.Error{}) {
		return httpserver.GetErrorHandler()(http.StatusBadRequest, err), nil
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
	out, err := ch.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return nil, err
	}

	return httpserver.NewJsonResponse(out), nil
}
