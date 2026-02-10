package db_repo

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type publisherNotifier struct {
	notifier
	publisher   Publisher
	transformer mdl.TransformerResolver
}

func NewPublisherNotifier(_ context.Context, config cfg.Config, publisher Publisher, logger log.Logger, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) (*publisherNotifier, error) {
	if err := modelId.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("can not pad model id from config: %w", err)
	}

	notifier := newNotifier(logger, modelId, version)

	return &publisherNotifier{
		notifier:    notifier,
		publisher:   publisher,
		transformer: transformer,
	}, nil
}

func (n *publisherNotifier) Send(ctx context.Context, notificationType string, value ModelBased) error {
	out := n.transformer("api", n.version, value)
	err := n.publisher.Publish(ctx, notificationType, n.version, out)

	if exec.IsRequestCanceled(err) {
		n.logger.Info(ctx, "request canceled while executing notification publish on %s for model %s with id %d", notificationType, n.modelId, value.GetId())
		n.writeMetric(ctx, err)

		return err
	}

	if err != nil {
		n.writeMetric(ctx, err)

		return fmt.Errorf("error executing notification on %s for model %s with id %d: %w", notificationType, n.modelId, *value.GetId(), err)
	}

	n.logger.Info(ctx, "published on %s successful, for model %s with id %d", notificationType, n.modelId, *value.GetId())
	n.writeMetric(ctx, nil)

	return nil
}
