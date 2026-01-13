package httpserver

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/smpl"
)

func SamplingMiddleware(ctx context.Context, config cfg.Config, logger log.Logger) (gin.HandlerFunc, error) {
	var err error
	var decider smpl.Decider

	if decider, err = smpl.ProvideDecider(ctx, config); err != nil {
		return nil, fmt.Errorf("could not initialize sampling decider: %w", err)
	}

	return func(ginCtx *gin.Context) {
		ctx = ginCtx.Request.Context()

		if smplCtx, _, err := decider.Decide(ctx); err != nil {
			logger.Warn(ctx, "could not decide on sampling: %s", err)
		} else {
			ginCtx.Request = ginCtx.Request.WithContext(smplCtx)
		}

		ginCtx.Request = ginCtx.Request.WithContext(ctx)
		ginCtx.Next()
	}, nil
}
