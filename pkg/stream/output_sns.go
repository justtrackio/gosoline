package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud"
	"github.com/applike/gosoline/pkg/exec"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sns"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/hashicorp/go-multierror"
)

type SnsOutputSettings struct {
	cfg.AppId
	TopicId string
	Client  cloud.ClientSettings
	Backoff exec.BackoffSettings
}

type snsOutput struct {
	logger mon.Logger
	tracer tracing.Tracer

	topic    sns.Topic
	settings SnsOutputSettings
}

func NewSnsOutput(config cfg.Config, logger mon.Logger, s SnsOutputSettings) Output {
	s.PadFromConfig(config)

	topic := sns.NewTopic(config, logger, &sns.Settings{
		AppId:   s.AppId,
		Client:  s.Client,
		Backoff: s.Backoff,
		TopicId: s.TopicId,
	})

	tracer := tracing.ProviderTracer(config, logger)

	return NewSnsOutputWithInterfaces(logger, tracer, topic, s)
}

func NewSnsOutputWithInterfaces(logger mon.Logger, tracer tracing.Tracer, topic sns.Topic, s SnsOutputSettings) Output {
	return &snsOutput{
		logger:   logger,
		tracer:   tracer,
		topic:    topic,
		settings: s,
	}
}

func (o *snsOutput) WriteOne(ctx context.Context, record *Message) error {
	return o.Write(ctx, []*Message{record})
}

func (o *snsOutput) Write(ctx context.Context, batch []*Message) error {
	spanName := fmt.Sprintf("sns-output-%v-%v-%v", o.settings.Family, o.settings.Application, o.settings.TopicId)

	ctx, trans := o.tracer.StartSubSpan(ctx, spanName)
	defer trans.Finish()

	return o.publishToTopic(ctx, batch)
}

func (o *snsOutput) publishToTopic(ctx context.Context, batch []*Message) error {
	errors := make([]error, 0)

	for _, msg := range batch {
		body, err := msg.MarshalToString()

		if err != nil {
			errors = append(errors, err)
			continue
		}

		err = o.topic.Publish(ctx, &body)

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
