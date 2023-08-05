package crud

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	dbRepo "github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type createHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]] struct {
	logger      log.Logger
	transformer CreateHandler[I, O, K, M]
	settings    Settings
}

func NewCreateHandler[I any, O any, K mdl.PossibleIdentifier, M dbRepo.ModelBased[K]](config cfg.Config, logger log.Logger, transformer CreateHandler[I, O, K, M]) (gin.HandlerFunc, error) {
	settings := Settings{}
	if err := config.UnmarshalKey(SettingsConfigKey, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal create handler settings: %w", err)
	}

	ch := createHandler[I, O, K, M]{
		logger:      logger,
		transformer: transformer,
		settings:    settings,
	}

	return httpserver.CreateJsonHandler(ch), nil
}

func (ch createHandler[I, O, K, M]) GetInput() any {
	var input I

	return &input
}

func (ch createHandler[I, O, K, M]) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	// replace context with a new one to prevent cancellations from client side
	// include a new timeout to ensure that requests will be cancelled
	ctx, cancel := exec.WithDelayedCancelContext(ctx, ch.settings.WriteTimeout)
	defer cancel()

	model, err := ch.transformer.TransformCreate(ctx, request.Body.(*I))
	if err != nil {
		return HandleErrorOnWrite(ctx, ch.logger, err)
	}

	repo := ch.transformer.GetRepository()
	err = repo.Create(ctx, model)
	if err != nil {
		return HandleErrorOnWrite(ctx, ch.logger, err)
	}

	reload, err := repo.Read(ctx, *model.GetId())
	if err != nil {
		return HandleErrorOnWrite(ctx, ch.logger, err)
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := ch.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return HandleErrorOnWrite(ctx, ch.logger, err)
	}

	return httpserver.NewJsonResponse(out), nil
}
