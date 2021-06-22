package crud

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	db_repo "github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/log"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
)

type readHandler struct {
	transformer BaseHandler
	logger      log.Logger
}

func NewReadHandler(logger log.Logger, transformer BaseHandler) gin.HandlerFunc {
	rh := readHandler{
		transformer: transformer,
		logger:      logger,
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

	var notFound db_repo.RecordNotFoundError
	if errors.As(err, &notFound) {
		rh.logger.WithContext(ctx).Warn("failed to read model: %s", err)
		return apiserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if err != nil {
		return nil, err
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := rh.transformer.TransformOutput(model, apiView)

	if err != nil {
		return nil, err
	}

	return apiserver.NewJsonResponse(out), nil
}
