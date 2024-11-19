package db_repo

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type publisherNotifier[K mdl.PossibleIdentifier] struct {
	notifier
	publisher   Publisher
	transformer mdl.TransformerResolver
}

func NewPublisherNotifier[K mdl.PossibleIdentifier](config cfg.Config, publisher Publisher, logger log.Logger, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) Notifier[K] {
	modelId.PadFromConfig(config)

	notifier := newNotifier(logger, modelId, version)

	return &publisherNotifier[K]{
		notifier:    notifier,
		publisher:   publisher,
		transformer: transformer,
	}
}

func (n *publisherNotifier[K]) Send(ctx context.Context, notificationType string, value ModelBased[K]) error {
	logger := n.logger.WithContext(ctx)

	out := n.transformer("api", n.version, value)
	err := n.publisher.Publish(ctx, notificationType, n.version, out)

	if exec.IsRequestCanceled(err) {
		logger.Info("request canceled while executing notification publish on %s for model %s with id %v", notificationType, n.modelId, value.GetId())
		n.writeMetric(err)

		return err
	}

	if err != nil {
		n.writeMetric(err)

		return fmt.Errorf("error executing notification on %s for model %s with id %v: %w", notificationType, n.modelId, *value.GetId(), err)
	}

	logger.Info("published on %s successful, for model %s with id %v", notificationType, n.modelId, *value.GetId())
	n.writeMetric(nil)

	return nil
}
