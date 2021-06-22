package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/log"
	"github.com/applike/gosoline/pkg/sns"
	"github.com/hashicorp/go-multierror"
)

type SnsOutputSettings struct {
	cfg.AppId
	TopicId string
	Client  cloud.ClientSettings
	Backoff exec.BackoffSettings
}

type snsOutput struct {
	logger   log.Logger
	topic    sns.Topic
	settings SnsOutputSettings
}

func NewSnsOutput(config cfg.Config, logger log.Logger, s SnsOutputSettings) (Output, error) {
	s.PadFromConfig(config)

	topic, err := sns.NewTopic(config, logger, &sns.Settings{
		AppId:   s.AppId,
		Client:  s.Client,
		Backoff: s.Backoff,
		TopicId: s.TopicId,
	})
	if err != nil {
		return nil, fmt.Errorf("can not create topic: %w", err)
	}

	return NewSnsOutputWithInterfaces(logger, topic, s), nil
}

func NewSnsOutputWithInterfaces(logger log.Logger, topic sns.Topic, s SnsOutputSettings) Output {
	return &snsOutput{
		logger:   logger,
		topic:    topic,
		settings: s,
	}
}

func (o *snsOutput) WriteOne(ctx context.Context, record WritableMessage) error {
	return o.Write(ctx, []WritableMessage{record})
}

func (o *snsOutput) Write(ctx context.Context, batch []WritableMessage) error {
	errors := make([]error, 0)

	for _, msg := range batch {
		body, err := msg.MarshalToString()

		if err != nil {
			errors = append(errors, err)
			continue
		}

		err = o.topic.Publish(ctx, &body, getAttributes(msg))

		if err != nil {
			errors = append(errors, err)
			continue
		}
	}

	if len(errors) == 1 {
		return errors[0]
	}

	if len(errors) > 0 {
		return &multierror.Error{
			Errors: errors,
		}
	}

	return nil
}
