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
	publisher Publisher
}

func NewPublisherNotifier(_ context.Context, config cfg.Config, publisher Publisher, logger log.Logger, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) *publisherNotifier {
	modelId.PadFromConfig(config)
	notifier := newNotifier(logger, modelId, version, transformer)

	return &publisherNotifier{
		notifier:  notifier,
		publisher: publisher,
	}
}

func (n *publisherNotifier) Send(ctx context.Context, notificationType string, value ModelBased) error {
	logger := n.logger.WithContext(ctx)

	out := n.transformer(TransformerDefaultView, n.version, value)
	err := n.publisher.Publish(ctx, notificationType, n.version, out)

	if exec.IsRequestCanceled(err) {
		logger.Info("request canceled while executing notification publish on %s for model %s with id %d", notificationType, n.modelId, value.GetId())
		n.writeMetric(err)
		return err
	}

	if err != nil {
		n.writeMetric(err)
		return fmt.Errorf("error executing notification on %s for model %s with id %d: %w", notificationType, n.modelId, *value.GetId(), err)
	}

	logger.Info("published on %s successful, for model %s with id %d", notificationType, n.modelId, *value.GetId())
	n.writeMetric(nil)

	return nil
}
