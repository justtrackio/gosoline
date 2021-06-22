package stream

import (
	"fmt"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/cloud/aws/kinesis"
	"github.com/applike/gosoline/pkg/encoding/json"
	"github.com/applike/gosoline/pkg/log"
)

type kinesisInput struct {
	kinesis.Reader
	channel chan *Message
}

func NewKinesisInput(config cfg.Config, logger log.Logger, factory kinesis.KinsumerFactory, settings kinesis.KinsumerSettings) (Input, error) {
	channel := make(chan *Message)
	sink := NewKinesisMessageHandler(channel)
	input, err := kinesis.NewReader(config, logger, factory, sink, settings)

	if err != nil {
		return nil, fmt.Errorf("failed to create kinsumer input: %w", err)
	}

	return &kinesisInput{
		Reader:  input,
		channel: channel,
	}, nil
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
