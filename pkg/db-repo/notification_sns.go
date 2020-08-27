package db_repo

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

func NewSnsNotifier(config cfg.Config, logger mon.Logger, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) *baseNotifier {
	output := stream.NewSnsOutput(config, logger, stream.SnsOutputSettings{
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

	return NewBaseNotifier(logger, output, modelId, version, transformer)
}
