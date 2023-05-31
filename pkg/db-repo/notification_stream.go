package db_repo

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/stream"
)

type streamNotifier struct {
	notifier
	encoder stream.MessageEncoder
	output  stream.Output
}

func NewStreamNotifier(logger log.Logger, output stream.Output, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) *streamNotifier {
	notifier := newNotifier(logger, modelId, version, transformer)

	encoder := stream.NewMessageEncoder(&stream.MessageEncoderSettings{
		Encoding: stream.EncodingJson,
	})

	return &streamNotifier{
		notifier: notifier,
		encoder:  encoder,
		output:   output,
	}
}

func (n *streamNotifier) Send(ctx context.Context, notificationType string, value ModelBased) error {
	logger := n.logger.WithContext(ctx)
	modelId := n.modelId.String()

	out := n.transformer("api", n.version, value)

	msg, err := n.encoder.Encode(ctx, out, map[string]interface{}{
		"type":    notificationType,
		"version": n.version,
		"modelId": modelId,
	})
	if err != nil {
		return fmt.Errorf("can not encode notification message: %w", err)
	}

	err = n.output.WriteOne(ctx, msg)

	if exec.IsRequestCanceled(err) {
		logger.Info("request canceled while executing notification on %s for model %s with id %d", notificationType, modelId, *value.GetId())
		n.writeMetric(err)
		return err
	}

	if err != nil {
		n.writeMetric(err)
		return fmt.Errorf("error executing notification on %s for model %s with id %d: %w", notificationType, modelId, *value.GetId(), err)
	}

	logger.Info("sent on %s successful for model %s with id %d", notificationType, modelId, *value.GetId())
	n.writeMetric(nil)

	return nil
}

func NewSnsNotifier(ctx context.Context, config cfg.Config, logger log.Logger, modelId mdl.ModelId, version int, transformer mdl.TransformerResolver) (*streamNotifier, error) {
	modelId.PadFromConfig(config)

	output, err := stream.NewSnsOutput(ctx, config, logger, &stream.SnsOutputSettings{
		TopicId: modelId.Name,
		AppId: cfg.AppId{
			Project:     modelId.Project,
			Environment: modelId.Environment,
			Family:      modelId.Family,
			Application: modelId.Application,
		},
		ClientName: "default",
	})
	if err != nil {
		return nil, fmt.Errorf("can not create sns output: %w", err)
	}

	return NewStreamNotifier(logger, output, modelId, version, transformer), nil
}
