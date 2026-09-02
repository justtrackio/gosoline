package stream

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/justtrackio/gosoline/pkg/log"
)

type kinesisInput struct {
	client kinesis.Kinsumer
}

var _ Input = &kinesisInput{}

func NewKinesisInput(ctx context.Context, config cfg.Config, logger log.Logger, settings kinesis.Settings) (Input, error) {
	client, err := kinesis.NewKinsumer(ctx, config, logger, &settings)
	if err != nil {
		return nil, fmt.Errorf("unable to create kinesis client: %w", err)
	}

	return &kinesisInput{
		client: client,
	}, nil
}

func (i *kinesisInput) Run(ctx context.Context, process InputProcess) error {
	return i.client.Run(ctx, func(ctx context.Context, rawMessage []byte) error {
		msg := Message{}
		if err := msg.UnmarshalFromBytes(rawMessage); err != nil {
			return fmt.Errorf("failed to unmarshal message: %w", err)
		}

		// Kinesis advances handled records like Kafka commits them; consumer retries are handled separately. Once
		// process returned, the record was handled and has to be checkpointed, no matter whether it was acknowledged
		// and no matter whether the context expired in the meantime: reporting a cancellation here would make the
		// shard reader stop checkpointing before this record, so it would be consumed again although the consumer
		// already put it into the retry queue.
		process(ctx, &msg)

		return nil
	})
}

func (i *kinesisInput) Stop(ctx context.Context) {
	i.client.Stop(ctx)
}

func (i *kinesisInput) IsHealthy() bool {
	return i.client.IsHealthy()
}
