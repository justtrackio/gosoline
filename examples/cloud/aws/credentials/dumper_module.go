package credentials

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sqs"
	"github.com/justtrackio/gosoline/pkg/encoding/yaml"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

func NewDebugModule(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
	_, err := sqs.NewClient(ctx, config, logger, "example")

	return &DebugModude{
		config: config.Get("cloud.aws"),
	}, err
}

type DebugModude struct {
	config any
}

func (d DebugModude) Run(ctx context.Context) error {
	config, err := yaml.Marshal(map[string]any{
		"cloud": map[string]any{
			"aws": d.config,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal aws config: %w", err)
	}

	fmt.Println(string(config))

	return nil
}
