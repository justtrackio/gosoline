package kinesis

import (
	"context"
	"fmt"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
)

// ShardTimeout describes the timeout until we remove records about shards from ddb.
// kinesis has a maximum retention time of 7 days, so we are safe to remove old data from ddb after 8 days (or 7 days and
// 1 second...). We need this to clean up the database if there are old shards for a stream (i.e., after a stream was scaled
// and the number of shards was reduced or a shard was replaced by different shards).
const ShardTimeout = time.Hour * 24 * 8

var (
	ErrCheckpointNoLongerOwned   = fmt.Errorf("can not persist checkpoint which is no longer owned")
	ErrCheckpointAlreadyReleased = fmt.Errorf("failed to release checkpoint, it was already released")
	ErrShardAlreadyFinished      = fmt.Errorf("shard was alread finished")
)

//go:generate mockery --name MetadataRepository
type MetadataRepository interface {
	// RegisterClient either creates or refreshes our registration and returns our current index as well as the number of
	// clients working together on the stream.
	RegisterClient(ctx context.Context) (clientIndex int, totalClients int, err error)
	// DeregisterClient removes our client and thus indirectly informs the other clients to take over our work
	DeregisterClient(ctx context.Context) error
	// AcquireShard locks a shard for us and ensures no other client is working on it. It might fail to do so and returns
	// nil in this case.
	AcquireShard(ctx context.Context, shardId ShardId) (Checkpoint, error)
}

// A Checkpoint describes our position in a shard of the stream.
//go:generate mockery --name Checkpoint
type Checkpoint interface {
	CheckpointWithoutRelease
	// Release releases ownership over a shard. Do not use the Checkpoint afterwards.
	Release(ctx context.Context) error
}

// CheckpointWithoutRelease consists of the Checkpoint interface without the release method. We only use this internally
// to ensure Release can only be called when we have taken ownership of the Checkpoint.
//go:generate mockery --name CheckpointWithoutRelease
type CheckpointWithoutRelease interface {
	GetSequenceNumber() SequenceNumber
	// Advance updates our Checkpoint to include all the sequence numbers up to (and including) the new sequence number.
	Advance(sequenceNumber SequenceNumber) error
	// Done marks a shard as completely consumed, i.e., there are no further records left to consume.
	Done(sequenceNumber SequenceNumber) error
	// Persist writes the current Checkpoint to the database and renews our lock on the shard. Thus, you have to call
	// Persist from time to time, otherwise you will lose your hold on that lock. If we finished the shard (by calling Done),
	// Persist will tell us to Release the Checkpoint (and shard).
	Persist(ctx context.Context) (shouldRelease bool, err error)
}

// A BaseRecord contains all the fields shared between the main table and all LSIs.
type BaseRecord struct {
	Namespace string    `json:"namespace" ddb:"key=hash"`
	Resource  string    `json:"resource" ddb:"key=range"`
	UpdatedAt time.Time `json:"updatedAt"`
	Ttl       *int64    `json:"ttl,omitempty" ddb:"ttl=enabled"`
}

type FullRecord struct {
	BaseRecord
	OwningClientId ClientId       `json:"owningClientId,omitempty"`
	SequenceNumber SequenceNumber `json:"sequenceNumber,omitempty"`
	FinishedAt     *time.Time     `json:"finishedAt"`
}

type ClientRecord struct {
	BaseRecord
}

type CheckpointRecord struct {
	BaseRecord
	OwningClientId ClientId       `json:"owningClientId,omitempty"`
	SequenceNumber SequenceNumber `json:"sequenceNumber,omitempty"`
	FinishedAt     *time.Time     `json:"finishedAt"`
}

type metadataRepository struct {
	logger            log.Logger
	stream            Stream
	clientId          ClientId
	repo              ddb.Repository
	appId             cfg.AppId
	clientTimeout     time.Duration
	checkpointTimeout time.Duration
	clock             clock.Clock
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

func NewMetadataRepository(ctx context.Context, config cfg.Config, logger log.Logger, stream Stream, clientId ClientId, settings Settings) (MetadataRepository, error) {
	ddbSettings := &ddb.Settings{
		ModelId: mdl.ModelId{
			Application: "kinsumer",
			Name:        "metadata",
		},
		Main: ddb.MainSettings{
			Model:              &FullRecord{},
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
		DisableTracing: true,
	}

	var err error
	var repo ddb.Repository

	if repo, err = ddb.NewRepository(ctx, config, logger, ddbSettings); err != nil {
		return nil, fmt.Errorf("can not create ddb repository: %w", err)
	}

	return NewMetadataRepositoryWithInterfaces(logger, stream, clientId, repo, settings, clock.Provider), nil
}

func NewMetadataRepositoryWithInterfaces(logger log.Logger, stream Stream, clientId ClientId, repo ddb.Repository, settings Settings, clock clock.Clock) MetadataRepository {
	clientTimeout := settings.DiscoverFrequency * 5
	if clientTimeout < time.Minute {
		clientTimeout = time.Minute
	}

	checkpointTimeout := settings.PersistFrequency * 5
	if checkpointTimeout < time.Minute {
		checkpointTimeout = time.Minute
	}

	return &metadataRepository{
		logger:            logger,
		stream:            stream,
		clientId:          clientId,
		repo:              repo,
		appId:             settings.AppId,
		clientTimeout:     clientTimeout,
		checkpointTimeout: checkpointTimeout,
		clock:             clock,
	}
}

func (m *metadataRepository) RegisterClient(ctx context.Context) (clientIndex int, totalClients int, err error) {
	namespace := m.getClientNamespace()
	_, err = m.repo.PutItem(ctx, m.repo.PutItemBuilder(), &ClientRecord{
		BaseRecord: BaseRecord{
			Namespace: namespace,
			Resource:  string(m.clientId),
			UpdatedAt: m.clock.Now(),
			Ttl:       mdl.Int64(m.clock.Now().Add(m.clientTimeout).Unix()),
		},
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to register client: %w", err)
	}

	qb := m.repo.QueryBuilder().WithHash(namespace)
	clients := make([]ClientRecord, 0)
	if _, err = m.repo.Query(ctx, qb, &clients); err != nil {
		return 0, 0, fmt.Errorf("failed to list clients: %w", err)
	}

	// find the index for the client when sorted by id (ddb returns items sorted by range by default)
	for i, client := range clients {
		if ClientId(client.Resource) == m.clientId {
			return i, len(clients), nil
		}
	}

	return 0, 0, fmt.Errorf("failed to find client just written to ddb")
}

func (m *metadataRepository) DeregisterClient(ctx context.Context) error {
	namespace := m.getClientNamespace()

	qb := m.repo.DeleteItemBuilder().
		WithHash(namespace).
		WithRange(m.clientId)
	if _, err := m.repo.DeleteItem(ctx, qb, &ClientRecord{}); err != nil {
		return fmt.Errorf("failed to deregister client: %w", err)
	}

	return nil
}

func (m *metadataRepository) AcquireShard(ctx context.Context, shardId ShardId) (Checkpoint, error) {
	timedOutBefore := m.clock.Now().Add(-m.checkpointTimeout)
	namespace := m.getCheckpointNamespace()

	getQb := m.repo.GetItemBuilder().
		WithHash(namespace).
		WithRange(shardId).
		WithConsistentRead(true)
	record := &CheckpointRecord{}
	getResult, err := m.repo.GetItem(ctx, getQb, record)
	if err != nil {
		return nil, fmt.Errorf("failed to read checkpoint record: %w", err)
	}

	if !getResult.IsFound {
		m.logger.Info("trying to use unused shard %s", shardId)

		record = &CheckpointRecord{
			BaseRecord: BaseRecord{
				Namespace: namespace,
				Resource:  string(shardId),
				UpdatedAt: m.clock.Now(),
				Ttl:       mdl.Int64(m.clock.Now().Add(ShardTimeout).Unix()),
			},
			OwningClientId: m.clientId,
			SequenceNumber: "",
			FinishedAt:     nil,
		}
	} else {
		if record.OwningClientId != "" && record.UpdatedAt.After(timedOutBefore) {
			m.logger.Info("not trying to take over shard %s from %s, it is still in use", shardId, record.OwningClientId)

			return nil, nil
		}

		if record.FinishedAt != nil {
			m.logger.Info("not trying to take over shard %s, it is already finished", shardId)

			return nil, ErrShardAlreadyFinished
		}

		owner := record.OwningClientId
		if owner == "" {
			owner = "nobody"
		}
		m.logger.Info("trying to take over shard %s from %s", shardId, owner)

		record.OwningClientId = m.clientId
		record.UpdatedAt = m.clock.Now()
		record.Ttl = mdl.Int64(m.clock.Now().Add(ShardTimeout).Unix())
	}

	putQb := m.repo.PutItemBuilder().
		WithCondition(ddb.AttributeNotExists("owningClientId").Or(ddb.Lte("updatedAt", timedOutBefore)))

	if putResult, err := m.repo.PutItem(ctx, putQb, record); err != nil {
		return nil, fmt.Errorf("failed to write checkpoint record: %w", err)
	} else if putResult.ConditionalCheckFailed {
		m.logger.Info("failed to acquire shard %s", shardId)

		return nil, nil
	}

	finalSequenceNumber := record.SequenceNumber
	if record.FinishedAt == nil {
		finalSequenceNumber = ""
	}

	return &checkpoint{
		repo:                m.repo,
		clock:               m.clock,
		lck:                 conc.NewPoisonedLock(),
		namespace:           namespace,
		shardId:             shardId,
		owningClientId:      m.clientId,
		sequenceNumber:      record.SequenceNumber,
		finalSequenceNumber: finalSequenceNumber,
		finishedAt:          record.FinishedAt,
	}, nil
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

	c.finishedAt = mdl.Time(c.clock.Now())
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
			Ttl:       mdl.Int64(c.clock.Now().Add(ShardTimeout).Unix()),
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
			Set("ttl", mdl.Int64(c.clock.Now().Add(ShardTimeout).Unix())).
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

func (m *metadataRepository) getClientNamespace() string {
	return fmt.Sprintf("client:%s:%s", m.appId.String(), m.stream)
}

func (m *metadataRepository) getCheckpointNamespace() string {
	return fmt.Sprintf("checkpoint:%s:%s", m.appId.String(), m.stream)
}
