package db_repo

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

func NewSnsNotifier(config cfg.Config, logger mon.Logger, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) *baseNotifier {
	output := stream.NewSnsOutput(config, logger, stream.SnsOutputSettings{
		TopicId: modelId.Name,
	})

	defaults := getDefaultNotifierMetrics(modelId)
	mtr := mon.NewMetricDaemonWriter(defaults...)

	return NewSnsNotifierWithInterfaces(logger, mtr, output, modelId, version, transformer)
}

func NewSnsNotifierWithInterfaces(logger mon.Logger, mtr mon.MetricWriter, output stream.Output, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) *baseNotifier {
	return &baseNotifier{
		logger:      logger,
		metric:      mtr,
		output:      output,
		modelId:     modelId,
		version:     version,
		transformer: transformer,
	}
}
