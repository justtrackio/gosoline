package crud

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/pkg/errors"
)

type readHandler struct {
	logger      log.Logger
	transformer BaseHandler
}

func NewReadHandler(logger log.Logger, transformer BaseHandler) gin.HandlerFunc {
	rh := readHandler{
		transformer: transformer,
		logger:      logger,
	}

	return httpserver.CreateHandler(rh)
}

func (rh readHandler) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	id, valid := httpserver.GetUintFromRequest(request, "id")

	if !valid {
		return nil, errors.New("no valid id provided")
	}

	repo := rh.transformer.GetRepository()
	model := rh.transformer.GetModel()
	err := repo.Read(ctx, id, model)

	var notFound db_repo.RecordNotFoundError
	if errors.As(err, &notFound) {
		rh.logger.WithContext(ctx).Warn("failed to read model: %s", err)

		return httpserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if err != nil {
		return nil, err
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := rh.transformer.TransformOutput(ctx, model, apiView)
	if err != nil {
		return nil, err
	}

	return httpserver.NewJsonResponse(out), nil
}
