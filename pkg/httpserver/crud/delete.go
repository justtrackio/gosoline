package crud

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type deleteHandler struct {
	logger      log.Logger
	transformer BaseHandler
	settings    Settings
}

func NewDeleteHandler(config cfg.Config, logger log.Logger, transformer BaseHandler) gin.HandlerFunc {
	settings := Settings{}
	config.UnmarshalKey(SettingsConfigKey, &settings)

	dh := deleteHandler{
		transformer: transformer,
		logger:      logger,
		settings:    settings,
	}

	return httpserver.CreateHandler(dh)
}

func (dh deleteHandler) Handle(reqCtx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	// replace context with a new one to prevent cancellations from client side
	// include a new timeout to ensure that requests will be cancelled
	ctx, cancel := exec.WithDelayedCancelContext(reqCtx, dh.settings.WriteTimeout)
	defer cancel()

	logger := dh.logger.WithContext(ctx)

	id, valid := httpserver.GetUintFromRequest(request, "id")

	if !valid {
		return handleErrorOnWrite(ctx, logger, &validation.Error{
			Errors: []error{
				errors.New("no valid id provided"),
			},
		})
	}

	repo := dh.transformer.GetRepository()
	model := dh.transformer.GetModel()

	err := repo.Read(ctx, id, model)
	if err != nil {
		return handleErrorOnWrite(ctx, logger, err)
	}

	err = repo.Delete(ctx, model)
	if err != nil {
		return handleErrorOnWrite(ctx, logger, err)
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := dh.transformer.TransformOutput(ctx, model, apiView)
	if err != nil {
		return handleErrorOnWrite(ctx, logger, err)
	}

	return httpserver.NewJsonResponse(out), nil
}
