package db_repo

import (
	"context"
	"fmt"

	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/stream"
)

func NewSnsNotifier(config cfg.Config, logger log.Logger, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) (*baseNotifier, error) {
	output, err := stream.NewSnsOutput(context.Background(), config, logger, &stream.SnsOutputSettings{
		TopicId: modelId.Name,
		AppId: cfg.AppId{
			Project:     modelId.Project,
			Environment: modelId.Environment,
			Family:      modelId.Family,
			Application: modelId.Application,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("can not create sns output: %w", err)
	}

	return NewBaseNotifier(logger, output, modelId, version, transformer), nil
}
