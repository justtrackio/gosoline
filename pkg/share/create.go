package share

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/db"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/crud"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type shareCreateHandler struct {
	logger       log.Logger
	transformer  ShareCreateHandler
	uuidProvider uuid.Uuid
}

func NewShareCreateHandler(logger log.Logger, transformer ShareCreateHandler) gin.HandlerFunc {
	sh := shareCreateHandler{
		logger:       logger,
		transformer:  transformer,
		uuidProvider: uuid.New(),
	}

	return httpserver.CreateJsonHandler(sh)
}

func (s shareCreateHandler) GetInput() any {
	return s.transformer.GetCreateInput()
}

func (s shareCreateHandler) Handle(ctx context.Context, req *httpserver.Request) (*httpserver.Response, error) {
	logger := s.logger.WithContext(ctx)

	id, valid := httpserver.GetUintFromRequest(req, "id")
	if !valid {
		return nil, errors.New("no valid id provided")
	}

	entity, err := s.getEntity(ctx, id)
	var notFound db_repo.RecordNotFoundError
	if errors.As(err, &notFound) {
		logger.Warn("failed to read entity: %s", err.Error())

		return httpserver.NewStatusResponse(http.StatusNotFound), nil
	}

	if err != nil {
		return nil, err
	}

	model := s.transformer.GetModel()
	// we assert cast safely here as the req.Body will get parsed in something that implements Metadata
	shareInput := req.Body.(Metadata)
	policy := BuildSharePolicy(s.uuidProvider.NewV4(), entity, shareInput.GetOwnerId(), shareInput.GetActions())

	guard := s.transformer.GetGuard()
	err = guard.CreatePolicy(ctx, policy)
	if err != nil {
		return nil, err
	}

	err = s.transformer.TransformCreate(ctx, req.Body, entity, policy, model)
	if err != nil {
		return nil, err
	}

	shareRepo := s.transformer.GetRepository()
	err = shareRepo.Create(ctx, model)

	if db.IsDuplicateEntryError(err) {
		return httpserver.NewStatusResponse(http.StatusConflict), nil
	}

	if errors.Is(err, &validation.Error{}) {
		return httpserver.GetErrorHandler()(http.StatusBadRequest, err), nil
	}

	if err != nil {
		return nil, err
	}

	reload := s.transformer.GetModel()
	err = shareRepo.Read(ctx, model.GetId(), reload)
	if err != nil {
		return nil, err
	}

	apiView := crud.GetApiViewFromHeader(req.Header)
	out, err := s.transformer.TransformOutput(ctx, reload, apiView)
	if err != nil {
		return nil, err
	}

	return httpserver.NewJsonResponse(out), nil
}

func (s shareCreateHandler) getEntity(ctx context.Context, id *uint) (Shareable, error) {
	entity := s.transformer.GetEntityModel()
	entityRepo := s.transformer.GetEntityRepository()

	err := entityRepo.Read(ctx, id, entity)
	if err != nil {
		return nil, err
	}

	return entity, nil
}
