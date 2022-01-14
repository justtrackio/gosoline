package db_repo

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/stream"
)

func NewSnsNotifier(ctx context.Context, config cfg.Config, logger log.Logger, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) (*baseNotifier, error) {
	appId := cfg.AppId{
		Project:     modelId.Project,
		Environment: modelId.Environment,
		Family:      modelId.Family,
		Application: modelId.Application,
	}
	appId.PadFromConfig(config)

	output, err := stream.NewSnsOutput(ctx, config, logger, &stream.SnsOutputSettings{
		TopicId: modelId.Name,
		AppId:   appId,
	})
	if err != nil {
		return nil, fmt.Errorf("can not create sns output: %w", err)
	}

	return NewBaseNotifier(logger, output, modelId, version, transformer), nil
}
