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
	config, _ := yaml.Marshal(map[string]interface{}{
		"cloud": map[string]interface{}{
			"aws": d.config,
		},
	})
	fmt.Println(string(config))

	return nil
}
