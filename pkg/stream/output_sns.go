package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/sns"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

type snsOutput struct {
	logger log.Logger
	topic  sns.Topic
}

func NewSnsOutput(ctx context.Context, config cfg.Config, logger log.Logger, settings *SnsOutputSettings) (Output, error) {
	settings.PadFromConfig(config)

	var err error
	var topic sns.Topic
	var topicName string

	if topicName, err = sns.GetTopicName(config, settings); err != nil {
		return nil, fmt.Errorf("can not get topic name: %w", err)
	}

	topicSettings := &sns.TopicSettings{
		TopicName:  topicName,
		ClientName: settings.ClientName,
	}

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

func (o *snsOutput) WriteOne(ctx context.Context, message WritableMessage) error {
	var err error
	var body string
	attributes := getAttributes(message)

	if body, err = message.MarshalToString(); err != nil {
		return fmt.Errorf("can not marshal message to string: %w", err)
	}

	if err = o.topic.Publish(ctx, body, attributes); err != nil {
		return fmt.Errorf("can not publish message: %w", err)
	}

	return nil
}

func (o *snsOutput) Write(ctx context.Context, batch []WritableMessage) error {
	messages, attributes, err := o.computeMessagesAttributes(batch)
	if err != nil {
		return fmt.Errorf("could not compute message attributes: %w", err)
	}

	if err = o.topic.PublishBatch(ctx, messages, attributes); err != nil {
		return fmt.Errorf("can not publish message batch: %w", err)
	}

	return nil
}

func (o *snsOutput) computeMessagesAttributes(batch []WritableMessage) ([]string, []map[string]string, error) {
	messages := make([]string, 0, len(batch))
	attributes := make([]map[string]string, 0, len(batch))

	for i := 0; i < len(batch); i++ {
		message, err := batch[i].MarshalToString()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to marshal message %d: %w", i, err)
		}

		messages = append(messages, message)
		attributes = append(attributes, getAttributes(batch[i]))
	}

	return messages, attributes, nil
}

func (o *snsOutput) GetMaxMessageSize() *int {
	return mdl.Box(256 * 1024)
}

func (o *snsOutput) GetMaxBatchSize() *int {
	return mdl.Box(10)
}
