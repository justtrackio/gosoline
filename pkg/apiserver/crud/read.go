package crud

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	db_repo "github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"net/http"
)

type readHandler struct {
	transformer BaseHandler
}

func NewReadHandler(transformer BaseHandler) gin.HandlerFunc {
	rh := readHandler{
		transformer: transformer,
	}

	return apiserver.CreateHandler(rh)
}

func (rh readHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	logger := mon.NewLogger()
	id, valid := apiserver.GetUintFromRequest(request, "id")

	if !valid {
		return nil, errors.New("no valid id provided")
	}

	repo := rh.transformer.GetRepository()
	model := rh.transformer.GetModel()
	err := repo.Read(ctx, id, model)

	var notFound db_repo.RecordNotFoundError
	if errors.As(err, &notFound) {
		logger.Warnf("failed to read model:%s", err)
		return apiserver.NewStatusResponse(http.StatusNoContent), nil
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
