package mdlsub_test

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/mdlsub/mocks"
)

func init() {
	mdlsub.AddOutput("mock", func(ctx context.Context, config cfg.Config, logger log.Logger, settings *mdlsub.SubscriberSettings, transformers mdlsub.VersionedModelTransformers) (map[int]mdlsub.Output, error) {
		outputs := make(map[int]mdlsub.Output)

		for version := range transformers {
			outputs[version] = new(mocks.Output)
		}

		return outputs, nil
	})
}
