package crud

import (
	"context"
	"errors"
	"github.com/applike/gosoline/pkg/apiserver"
	db_repo "github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/validation"
	"github.com/gin-gonic/gin"
	"net/http"
)

type deleteHandler struct {
	transformer BaseHandler
	logger      log.Logger
}

func NewDeleteHandler(logger log.Logger, transformer BaseHandler) gin.HandlerFunc {
	dh := deleteHandler{
		transformer: transformer,
		logger:      logger,
	}

	return apiserver.CreateHandler(dh)
}

func (dh deleteHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	id, valid := apiserver.GetUintFromRequest(request, "id")

	if !valid {
		return nil, errors.New("no valid id provided")
	}

	repo := dh.transformer.GetRepository()
	model := dh.transformer.GetModel()

	err := repo.Read(ctx, id, model)

	var notFound db_repo.RecordNotFoundError
	if errors.As(err, &notFound) {
		dh.logger.WithContext(ctx).Warn("failed to delete model: %s", err)
		return apiserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if err != nil {
		return nil, err
	}

	err = repo.Delete(ctx, model)

	if errors.Is(err, &validation.Error{}) {
		return apiserver.GetErrorHandler()(http.StatusBadRequest, err), nil
	}

	if err != nil {
		return nil, err
	}

	apiView := GetApiViewFromHeader(request.Header)
	out, err := dh.transformer.TransformOutput(model, apiView)

	if err != nil {
		return nil, err
	}

	return apiserver.NewJsonResponse(out), nil
}
