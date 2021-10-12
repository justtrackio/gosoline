package uniqueid

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/apiserver"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

type NextIdResponse struct {
	Id int64 `json:"id"`
}

func DefineApi(ctx context.Context, config cfg.Config, logger log.Logger) (*apiserver.Definitions, error) {
	h, err := NewHandler(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create handler: %w", err)
	}

	d := &apiserver.Definitions{}

	d.GET("/nextId", apiserver.CreateHandler(h))

	return d, nil
}

type handler struct {
	logger    log.Logger
	generator Generator
}

func NewHandler(ctx context.Context, config cfg.Config, logger log.Logger) (*handler, error) {
	logger = logger.WithChannel("unique-id")

	generator, err := NewGeneratorSonyFlake(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("could not create generator: %w", err)
	}

	return &handler{
		logger:    logger,
		generator: generator,
	}, nil
}

func (h *handler) Handle(ctx context.Context, _ *apiserver.Request) (*apiserver.Response, error) {
	logger := h.logger.WithContext(ctx)

	id, err := h.generator.NextId(ctx)
	if err != nil {
		logger.Error("could not generate id: %w", err)

		return nil, fmt.Errorf("could not generate id: %w", err)
	}

	out := NextIdResponse{
		Id: *id,
	}

	return apiserver.NewJsonResponse(out), nil
}
