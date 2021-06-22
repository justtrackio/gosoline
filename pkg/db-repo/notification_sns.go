package db_repo

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/stream"
)

func NewSnsNotifier(config cfg.Config, logger log.Logger, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) (*baseNotifier, error) {
	output, err := stream.NewSnsOutput(config, logger, stream.SnsOutputSettings{
		TopicId: modelId.Name,
		AppId: cfg.AppId{
			Project:     modelId.Project,
			Environment: modelId.Environment,
			Family:      modelId.Family,
			Application: modelId.Application,
		},
		Backoff: exec.BackoffSettings{
			Enabled:  true,
			Blocking: true,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("can not create sns output: %w", err)
	}

	return NewBaseNotifier(logger, output, modelId, version, transformer), nil
}
