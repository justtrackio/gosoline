package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/coffin"
)

type consumerBatchAdapter struct {
	ConsumerCallback
}

type parallelConsumerBatchAdapter struct {
	ConsumerCallback
}

type consumerAdapter struct {
	BatchConsumerCallback
}

var (
	_ BatchConsumerCallback = consumerBatchAdapter{}
	_ BatchConsumerCallback = parallelConsumerBatchAdapter{}
	_ ConsumerCallback      = consumerAdapter{}
)

func (c consumerBatchAdapter) Consume(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
	acks := make([]bool, len(models))
	for i, model := range models {
		ack, err := c.ConsumerCallback.Consume(ctx, model, attributes[i])
		if err != nil {
			return acks, err
		}

		acks[i] = ack
	}

	return acks, nil
}

func (c parallelConsumerBatchAdapter) Consume(ctx context.Context, models []any, attributes []map[string]string) ([]bool, error) {
	acks := make([]bool, len(models))
	cfn := coffin.New()

	cfn.Go(func() error {
		for i, model := range models {
			i := i
			model := model

			cfn.Go(func() error {
				ack, err := c.ConsumerCallback.Consume(ctx, model, attributes[i])
				if err != nil {
					return err
				}

				acks[i] = ack

				return nil
			})
		}

		return nil
	})

	err := cfn.Wait()

	return acks, err
}

func (c consumerAdapter) Consume(ctx context.Context, model any, attributes map[string]string) (bool, error) {
	acks, err := c.BatchConsumerCallback.Consume(ctx, []any{model}, []map[string]string{attributes})
	if len(acks) != 1 {
		return false, err
	}

	return acks[0], err
}

// ConsumerToBatchConsumer turns any consumer callback to a batch consumer callback by calling the callback
// for each single item of the batch. All items are processed serially and acknowledged in one single call
// (useful if you want to save on calls to the underlying input without complicating your code).
//
// Don't forget to configure stream.consumer.<name>.batch_size to take advantage of batching.
func ConsumerToBatchConsumer(consumer ConsumerCallback) BatchConsumerCallback {
	return consumerBatchAdapter{
		ConsumerCallback: consumer,
	}
}

// ConsumerToParallelBatchConsumer is similar to ConsumerToBatchConsumer, but runs the callback in parallel
// for each item in the batch.
//
// Don't forget to configure stream.consumer.<name>.batch_size to take advantage of batching.
func ConsumerToParallelBatchConsumer(consumer ConsumerCallback) BatchConsumerCallback {
	return parallelConsumerBatchAdapter{
		ConsumerCallback: consumer,
	}
}

// BatchConsumerToConsumer turns a batch consumer to a normal consumer. It calls the batch consumer with
// a single item each time, allowing you to reuse the code of a batch consumer as a normal consumer.
func BatchConsumerToConsumer(consumer BatchConsumerCallback) ConsumerCallback {
	return consumerAdapter{
		BatchConsumerCallback: consumer,
	}
}
