package definitions

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type module struct {
	logger     log.Logger
	repository db_repo.Repository
	input      stream.Input
}

func ModuleFactory(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	repository, err := NewRepository(ctx, config, logger)
	if err != nil {
		return nil, fmt.Errorf("cannot create bid repository: %w", err)
	}

	input, err := stream.NewConfigurableInput(ctx, config, logger, "default")
	if err != nil {
		return nil, fmt.Errorf("cannot create input: %w", err)
	}

	return &module{
		logger:     logger,
		repository: repository,
		input:      input,
	}, nil
}

func (m module) Run(ctx context.Context) error {
	err := m.input.Run(ctx)
	if err != nil {
		return fmt.Errorf("cannot create run input: %w", err)
	}

	for {
		select {
		case msg, ok := <-m.input.Data():
			if !ok {
				return nil
			}

			m.processMessage(ctx, msg.Body)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (m module) processMessage(ctx context.Context, msg string) {
	logger := m.logger.WithContext(ctx)

	items := make([]Item, 0)
	if err := json.Unmarshal([]byte(msg), &items); err != nil {
		logger.Error("unable to unmarshall items: %w", err)
	}

	for _, item := range items {
		switch item.Action {
		case "create":
			if err := m.repository.Create(ctx, &item); err != nil {
				logger.Error("unable to create item: %w", err)
			}

		case "update":
			if err := m.repository.Update(ctx, &item); err != nil {
				logger.Error("unable to update item: %w", err)
			}
		case "delete":
			if err := m.repository.Delete(ctx, &item); err != nil {
				logger.Error("unable to delete item: %w", err)
			}
		default:
			logger.Error("unknown action %s", item.Action)
		}
	}
}
