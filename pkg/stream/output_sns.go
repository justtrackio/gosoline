package stream

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type SnsOutputSettings struct {
	cfg.AppId
	TopicId    string
	ClientName string
}

type snsOutput struct {
	logger log.Logger
	topic  sns.Topic
}

func NewSnsOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *SnsOutputSettings) (Output, error) {
	settings.PadFromConfig(config)

	topicName := sns.GetTopicName(settings.AppId, settings.TopicId)
	topicSettings := &sns.TopicSettings{
		TopicName:  topicName,
		ClientName: settings.ClientName,
	}

	var err error
	var topic sns.Topic

	if topic, err = sns.NewTopic(ctx, config, logger, topicSettings); err != nil {
		return nil, fmt.Errorf("can not create topic: %w", err)
	}

	return NewSnsOutputWithInterfaces(logger, topic), nil
}

func NewSnsOutputWithInterfaces(logger log.Logger, topic sns.Topic) Output {
	return &snsOutput{
		logger: logger,
		topic:  topic,
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

		err = o.topic.Publish(ctx, body, getAttributes(msg))

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

func (o *snsOutput) GetMaxMessageSize() *int {
	return mdl.Int(256 * 1024)
}

func (o *snsOutput) GetMaxBatchSize() *int {
	return mdl.Int(10)
}
