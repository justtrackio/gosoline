package stream

import (
	"context"
	"crypto/md5"
	"fmt"
	"math/big"
	"math/rand"

	"github.com/justtrackio/gosoline/pkg/log"
)

//go:generate mockery --name PartitionerRand
type PartitionerRand interface {
	Intn(n int) int
}

type producerDaemonPartitionedAggregator struct {
	logger      log.Logger
	rand        PartitionerRand
	buckets     []producerDaemonPartitionedAggregatorBucket
	bucketCount *big.Int
}

type producerDaemonPartitionedAggregatorBucket struct {
	aggregator ProducerDaemonAggregator
}

func NewProducerDaemonPartitionedAggregator(logger log.Logger, settings ProducerDaemonSettings, compression CompressionType) (ProducerDaemonAggregator, error) {
	partitionerRand := rand.New(rand.NewSource(int64(rand.Uint64())))
	createAggregator := func(attributes map[string]string) (ProducerDaemonAggregator, error) {
		return NewProducerDaemonAggregator(settings, compression, attributes)
	}

	return NewProducerDaemonPartitionedAggregatorWithInterfaces(logger, partitionerRand, settings.PartitionBucketCount, createAggregator)
}

func NewProducerDaemonPartitionedAggregatorWithInterfaces(logger log.Logger, rand PartitionerRand, aggregators int, createAggregator func(attributes map[string]string) (ProducerDaemonAggregator, error)) (ProducerDaemonAggregator, error) {
	buckets := make([]producerDaemonPartitionedAggregatorBucket, aggregators)
	bucketCount := big.NewInt(int64(len(buckets)))

	// compute (2^128 - 1) / bucketCount
	incrementStep := big.NewInt(1)
	incrementStep = incrementStep.Lsh(incrementStep, 128)
	incrementStep = incrementStep.Sub(incrementStep, big.NewInt(1))
	incrementStep = incrementStep.Div(incrementStep, bucketCount)
	// and half of that
	incrementStepHalf := (&big.Int{}).Div(incrementStep, big.NewInt(2))

	for bucket := range buckets {
		// compute incrementStep * bucket + 0.5 * incrementStep
		explicitHashKeyInt := big.NewInt(int64(bucket))
		explicitHashKeyInt = explicitHashKeyInt.Mul(explicitHashKeyInt, incrementStep)
		explicitHashKeyInt = explicitHashKeyInt.Add(explicitHashKeyInt, incrementStepHalf)

		aggregator, err := createAggregator(map[string]string{
			AttributeKinesisExplicitHashKey: explicitHashKeyInt.String(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create aggregator for bucket: %w", err)
		}

		buckets[bucket] = producerDaemonPartitionedAggregatorBucket{
			aggregator: aggregator,
		}
	}

	return &producerDaemonPartitionedAggregator{
		logger:      logger,
		rand:        rand,
		buckets:     buckets,
		bucketCount: bucketCount,
	}, nil
}

func (a *producerDaemonPartitionedAggregator) Write(ctx context.Context, msg *Message) ([]AggregateFlush, error) {
	explicitHashKey, err := a.getExplicitHashKeyForMessage(msg)
	if err != nil {
		a.logger.WithContext(ctx).Error("failed to determine partition or explicit hash key, will choose one at random: %w", err)
	}
	var bucketIndex int
	if explicitHashKey != nil {
		bucketIndex = int(big.NewInt(0).Mod(explicitHashKey, a.bucketCount).Int64())
	} else {
		bucketIndex = a.rand.Intn(len(a.buckets))
	}

	return a.buckets[bucketIndex].aggregator.Write(ctx, msg)
}

func (a *producerDaemonPartitionedAggregator) Flush() ([]AggregateFlush, error) {
	result := make([]AggregateFlush, 0)

	for _, bucket := range a.buckets {
		if flush, err := bucket.aggregator.Flush(); err != nil {
			return nil, fmt.Errorf("failed to flush bucket: %w", err)
		} else {
			result = append(result, flush...)
		}
	}

	return result, nil
}

func (a *producerDaemonPartitionedAggregator) getExplicitHashKeyForMessage(msg *Message) (*big.Int, error) {
	if explicitHashKeyString, ok := msg.Attributes[AttributeKinesisExplicitHashKey]; ok {
		if explicitHashKey, ok := big.NewInt(0).SetString(explicitHashKeyString, 10); !ok {
			return nil, fmt.Errorf("invalid explicit hash key: %s", explicitHashKeyString)
		} else {
			return explicitHashKey, nil
		}
	}

	if partitionKey, ok := msg.Attributes[AttributeKinesisPartitionKey]; ok {
		partitionKeyHash := md5.Sum([]byte(partitionKey))

		return big.NewInt(0).SetBytes(partitionKeyHash[:]), nil
	}

	return nil, nil
}
