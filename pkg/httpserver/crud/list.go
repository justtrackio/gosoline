package crud

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/httpserver"
	"github.com/justtrackio/gosoline/pkg/httpserver/sql"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/validation"
)

type Output struct {
	Total   int `json:"total"`
	Results any `json:"results"`
}

type listHandler struct {
	transformer ListHandler
	logger      log.Logger
}

func NewListHandler(_ cfg.Config, logger log.Logger, transformer ListHandler) gin.HandlerFunc {
	lh := listHandler{
		transformer: transformer,
		logger:      logger,
	}

	return httpserver.CreateJsonHandler(lh)
}

func (lh listHandler) GetInput() any {
	return sql.NewInput()
}

func (lh listHandler) Handle(ctx context.Context, request *httpserver.Request) (*httpserver.Response, error) {
	inp := request.Body.(*sql.Input)

	logger := lh.logger.WithContext(ctx)

	repo := lh.transformer.GetRepository()
	metadata := repo.GetMetadata()

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)
	if err != nil {
		return handleErrorOnRead(logger, &validation.Error{
			Errors: []error{err},
		})
	}

	apiView := GetApiViewFromHeader(request.Header)
	results, err := lh.transformer.List(ctx, qb, apiView)
	if err != nil {
		return handleErrorOnRead(logger, err)
	}

	model := lh.transformer.GetModel()
	total, err := repo.Count(ctx, qb, model)
	if err != nil {
		return handleErrorOnRead(logger, err)
	}

	out := Output{
		Total:   total,
		Results: results,
	}

	resp := httpserver.NewJsonResponse(out)
	resp.AddHeader(httpserver.ApiViewKey, apiView)

	return resp, nil
}
