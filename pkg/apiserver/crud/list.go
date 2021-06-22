package crud

import (
	"context"
	"github.com/applike/gosoline/pkg/apiserver"
	"github.com/applike/gosoline/pkg/apiserver/sql"
	"github.com/applike/gosoline/pkg/log"
	"github.com/gin-gonic/gin"
)

type Output struct {
	Total   int         `json:"total"`
	Results interface{} `json:"results"`
}

type listHandler struct {
	transformer ListHandler
	logger      log.Logger
}

func NewListHandler(logger log.Logger, transformer ListHandler) gin.HandlerFunc {
	lh := listHandler{
		transformer: transformer,
		logger:      logger,
	}

	return apiserver.CreateJsonHandler(lh)
}

func (lh listHandler) GetInput() interface{} {
	return sql.NewInput()
}

func (lh listHandler) Handle(ctx context.Context, request *apiserver.Request) (*apiserver.Response, error) {
	inp := request.Body.(*sql.Input)

	repo := lh.transformer.GetRepository()
	metadata := repo.GetMetadata()

	lqb := sql.NewOrmQueryBuilder(metadata)
	qb, err := lqb.Build(inp)

	if err != nil {
		return nil, err
	}

	apiView := GetApiViewFromHeader(request.Header)
	results, err := lh.transformer.List(ctx, qb, apiView)

	if err != nil {
		return nil, err
	}

	model := lh.transformer.GetModel()
	total, err := repo.Count(ctx, qb, model)

	if err != nil {
		return nil, err
	}

	out := Output{
		Total:   total,
		Results: results,
	}

	resp := apiserver.NewJsonResponse(out)
	resp.AddHeader(apiserver.ApiViewKey, apiView)

	return resp, nil
}
