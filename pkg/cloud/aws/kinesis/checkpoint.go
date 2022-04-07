package kinesis

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

// checkpointWrapper is used with atomic.Value. It allows us to store different types of the same interface in it without a panic.
type checkpointWrapper struct {
	Checkpoint
}

// once we replace the checkpoint value, we use this type which doesn't care and never fails if we release it twice
type nopCheckpoint struct{}

func (c nopCheckpoint) GetSequenceNumber() SequenceNumber {
	return ""
}

func (c nopCheckpoint) Advance(_ SequenceNumber) error {
	return nil
}

func (c nopCheckpoint) Done(_ SequenceNumber) error {
	return nil
}

func (c nopCheckpoint) Persist(_ context.Context) (shouldRelease bool, err error) {
	return false, nil
}

func (c nopCheckpoint) Release(_ context.Context) error {
	return nil
}

type checkpoint struct {
	repo                ddb.Repository
	clock               clock.Clock
	lck                 conc.PoisonedLock
	namespace           string
	shardId             ShardId
	owningClientId      ClientId
	sequenceNumber      SequenceNumber
	finalSequenceNumber SequenceNumber
	finishedAt          *time.Time
}

func (c *checkpoint) GetSequenceNumber() SequenceNumber {
	return c.sequenceNumber
}

func (c *checkpoint) Advance(sequenceNumber SequenceNumber) error {
	if err := c.lck.TryLock(); err != nil {
		return fmt.Errorf("can not advance already released checkpoint: %w", err)
	}
	defer c.lck.Unlock()

	c.sequenceNumber = sequenceNumber

	return nil
}

func (c *checkpoint) Done(sequenceNumber SequenceNumber) error {
	if err := c.lck.TryLock(); err != nil {
		return fmt.Errorf("can not mark already released checkpoint as done: %w", err)
	}
	defer c.lck.Unlock()

	c.finishedAt = mdl.Box(c.clock.Now())
	c.finalSequenceNumber = sequenceNumber

	return nil
}

func (c *checkpoint) Persist(ctx context.Context) (shouldRelease bool, err error) {
	if err := c.lck.TryLock(); err != nil {
		return false, fmt.Errorf("can not persist already released checkpoint: %w", err)
	}
	defer c.lck.Unlock()

	record := &CheckpointRecord{
		BaseRecord: BaseRecord{
			Namespace: c.namespace,
			Resource:  string(c.shardId),
			UpdatedAt: c.clock.Now(),
			Ttl:       mdl.Box(c.clock.Now().Add(ShardTimeout).Unix()),
		},
		OwningClientId: c.owningClientId,
		SequenceNumber: c.sequenceNumber,
		FinishedAt:     c.finishedAt,
	}

	if c.sequenceNumber != c.finalSequenceNumber && c.finalSequenceNumber != "" {
		record.FinishedAt = nil
	}

	qb := c.repo.PutItemBuilder().WithCondition(ddb.Eq("owningClientId", c.owningClientId))

	if result, err := c.repo.PutItem(ctx, qb, record); err != nil {
		return false, fmt.Errorf("failed to persist checkpoint: %w", err)
	} else if result.ConditionalCheckFailed {
		return false, ErrCheckpointNoLongerOwned
	}

	return record.FinishedAt != nil, nil
}

func (c *checkpoint) Release(ctx context.Context) error {
	return c.lck.PoisonIf(func() (bool, error) {
		qb := c.repo.UpdateItemBuilder().
			WithHash(c.namespace).
			WithRange(c.shardId).
			Remove("owningClientId").
			Set("updatedAt", c.clock.Now()).
			Set("ttl", mdl.Box(c.clock.Now().Add(ShardTimeout).Unix())).
			Set("sequenceNumber", c.sequenceNumber).
			WithCondition(ddb.Eq("owningClientId", c.owningClientId))
		if result, err := c.repo.UpdateItem(ctx, qb, &CheckpointRecord{}); err != nil {
			return false, fmt.Errorf("failed to release checkpoint: %w", err)
		} else if result.ConditionalCheckFailed {
			return true, ErrCheckpointAlreadyReleased
		}

		return true, nil
	})
}
