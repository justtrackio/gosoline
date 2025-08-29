package crud

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/validation"
	"github.com/pkg/errors"
)

type readHandler struct {
	logger      log.Logger
	transformer BaseHandler
}

func NewReadHandler(_ cfg.Config, logger log.Logger, transformer BaseHandler) gin.HandlerFunc {
	rh := readHandler{
		transformer: transformer,
		logger:      logger,
	}

	return httpserver.CreateHandler(rh)
}

func (rh readHandler) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	id, valid := httpserver.GetUintFromRequest(request, "id")

	if !valid {
		return HandleErrorOnRead(ctx, rh.logger, &validation.Error{
			Errors: []error{
				errors.New("no valid id provided"),
			},
		})
	}

	logger := rh.logger.WithFields(log.Fields{
		"entity_id": id,
	})

	repo := rh.transformer.GetRepository()
	model := rh.transformer.GetModel()
	err := repo.Read(ctx, id, model)
	if err != nil {
		return HandleErrorOnRead(ctx, logger, err)
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := rh.transformer.TransformOutput(ctx, model, apiView)
	if err != nil {
		return HandleErrorOnRead(ctx, logger, err)
	}

	return httpserver.NewJsonResponse(out), nil
}
