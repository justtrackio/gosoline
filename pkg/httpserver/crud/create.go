package crud

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
)

type createHandler struct {
	logger      log.Logger
	transformer CreateHandler
	settings    Settings
}

func NewCreateHandler(config cfg.Config, logger log.Logger, transformer CreateHandler) gin.HandlerFunc {
	settings := Settings{}
	config.UnmarshalKey(SettingsConfigKey, &settings)

	ch := createHandler{
		transformer: transformer,
		logger:      logger,
		settings:    settings,
	}

	return httpserver.CreateJsonHandler(ch)
}

func (ch createHandler) GetInput() any {
	return ch.transformer.GetCreateInput()
}

func (ch createHandler) Handle(reqCtx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	// replace context with a new one to prevent cancellations from client side
	// include a new timeout to ensure that requests will be cancelled
	ctx, cancel := exec.WithDelayedCancelContext(reqCtx, ch.settings.WriteTimeout)
	defer cancel()

	model := ch.transformer.GetModel()
	err := ch.transformer.TransformCreate(ctx, request.Body, model)
	if err != nil {
		return handleErrorOnWrite(ctx, ch.logger, err)
	}

	repo := ch.transformer.GetRepository()
	err = repo.Create(ctx, model)
	if err != nil {
		return handleErrorOnWrite(ctx, ch.logger, err)
	}

	reload := ch.transformer.GetModel()
	err = repo.Read(ctx, model.GetId(), reload)
	if err != nil {
		return handleErrorOnWrite(ctx, ch.logger, err)
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := ch.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return handleErrorOnWrite(ctx, ch.logger, err)
	}

	return httpserver.NewJsonResponse(out), nil
}
