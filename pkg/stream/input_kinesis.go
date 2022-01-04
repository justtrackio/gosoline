package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/justtrackio/gosoline/pkg/encoding/json"
	"github.com/justtrackio/gosoline/pkg/log"
)

type kinesisInput struct {
	client  kinesis.Kinsumer
	channel chan *Message
}

func NewKinesisInput(ctx context.Context, config cfg.Config, logger log.Logger, settings kinesis.Settings) (Input, error) {
	client, err := kinesis.NewKinsumer(ctx, config, logger, &settings)
	if err != nil {
		return nil, fmt.Errorf("unable to create kinesis client: %w", err)
	}

	return &kinesisInput{
		client:  client,
		channel: make(chan *Message),
	}, nil
}

func (i *kinesisInput) Run(ctx context.Context) error {
	return i.client.Run(ctx, NewKinesisMessageHandler(i.channel))
}

func (i *kinesisInput) Stop() {
	i.client.Stop()
}

func (i *kinesisInput) Data() chan *Message {
	return i.channel
}

type kinesisMessageHandler struct {
	channel chan *Message
}

func NewKinesisMessageHandler(channel chan *Message) kinesis.MessageHandler {
	return kinesisMessageHandler{
		channel: channel,
	}
}

func (s kinesisMessageHandler) Handle(rawMessage []byte) error {
	msg := Message{}
	err := json.Unmarshal(rawMessage, &msg)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	s.channel <- &msg

	return nil
}

func (s kinesisMessageHandler) Done() {
	close(s.channel)
}
