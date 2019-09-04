package stream

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/sns"
	"github.com/applike/gosoline/pkg/tracing"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type SnsOutputSettings struct {
	cfg.AppId
	TopicId string
}

type snsOutput struct {
	logger mon.Logger
	tracer tracing.Tracer

	topic    sns.Topic
	settings SnsOutputSettings
}

func NewSnsOutput(config cfg.Config, logger mon.Logger, s SnsOutputSettings) Output {
	s.PadFromConfig(config)

	topic := sns.NewTopic(config, logger, sns.Settings{
		AppId:   s.AppId,
		TopicId: s.TopicId,
	})

	tracer := tracing.NewAwsTracer(config)

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

	ctx, trans := o.tracer.StartSpanFromContext(ctx, spanName)
	defer trans.Finish()

	for _, msg := range batch {
		msg.Trace = trans.GetTrace()
	}

	return o.publishToTopic(ctx, batch)
}

func (o *snsOutput) publishToTopic(ctx context.Context, batch []*Message) error {
	var result error

	for _, msg := range batch {
		body, err := msg.MarshalToString()

		if err != nil {
			result = multierror.Append(result, err)

			continue
		}

		err = o.topic.Publish(ctx, &body)

		if err != nil {
			result = multierror.Append(result, err)

			continue
		}
	}

	if result != nil {
		return errors.Wrap(result, "there were errors during publishing to the topic")
	}

	return nil
}
