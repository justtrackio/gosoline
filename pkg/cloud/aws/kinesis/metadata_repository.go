package kinesis

import (
	"context"
	"fmt"
	"sync"
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
	// IsShardFinished checks if a shard has already been finished. It might cache this status as normally a shard, once
	// finished, should never change that status again.
	IsShardFinished(ctx context.Context, shardId ShardId) (bool, error)
	// AcquireShard locks a shard for us and ensures no other client is working on it. It might fail to do so and returns
	// nil in this case.
	AcquireShard(ctx context.Context, shardId ShardId) (Checkpoint, error)
}

// A Checkpoint describes our position in a shard of the stream.
//
//go:generate mockery --name Checkpoint
type Checkpoint interface {
	CheckpointWithoutRelease
	// Release releases ownership over a shard. Do not use the Checkpoint afterward.
	Release(ctx context.Context) error
}

// CheckpointWithoutRelease consists of the Checkpoint interface without the release method. We only use this internally
// to ensure Release can only be called when we have taken ownership of the Checkpoint.
//
//go:generate mockery --name CheckpointWithoutRelease
type CheckpointWithoutRelease interface {
	GetSequenceNumber() SequenceNumber
	GetShardIterator() ShardIterator
	// Advance updates our Checkpoint to include all the sequence numbers up to (and including) the new sequence number.
	Advance(sequenceNumber SequenceNumber, shardIterator ShardIterator) error
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
	OwningClientId    ClientId       `json:"owningClientId,omitempty"`
	SequenceNumber    SequenceNumber `json:"sequenceNumber,omitempty"`
	LastShardIterator ShardIterator  `json:"lastShardIterator,omitempty"`
	FinishedAt        *time.Time     `json:"finishedAt"`
}

type ClientRecord struct {
	BaseRecord
}

type CheckpointRecord struct {
	BaseRecord
	OwningClientId    ClientId       `json:"owningClientId,omitempty"`
	SequenceNumber    SequenceNumber `json:"sequenceNumber,omitempty"`
	LastShardIterator ShardIterator  `json:"lastShardIterator,omitempty"`
	FinishedAt        *time.Time     `json:"finishedAt"`
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
	// an append-only map of finished shards. While we can never (besides restarting the app) remove something from this
	// map, a finished shard will stay so forever, shard ids are not reused, and an AWS account is limited by default to
	// 500 shards - so even if you have a lot of shards, you will most likely only need a few KB for this map.
	finishedMap map[ShardId]bool
	finishedLck sync.Mutex
}

func NewMetadataRepository(ctx context.Context, config cfg.Config, logger log.Logger, stream Stream, clientId ClientId, settings Settings) (MetadataRepository, error) {
	ddbSettings := &ddb.Settings{
		ClientName: settings.ClientName,
		ModelId: mdl.ModelId{
			Group: "kinsumer",
			Name:  "metadata",
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

	// we need the app id from the application we are running at, not the app id from the settings as this is the same
	// for different kinsumers of the same stream!
	appId := cfg.AppId{}
	appId.PadFromConfig(config)

	return NewMetadataRepositoryWithInterfaces(logger, stream, clientId, repo, settings, appId, clock.Provider), nil
}

func NewMetadataRepositoryWithInterfaces(logger log.Logger, stream Stream, clientId ClientId, repo ddb.Repository, settings Settings, appId cfg.AppId, clock clock.Clock) MetadataRepository {
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
		appId:             appId,
		clientTimeout:     clientTimeout,
		checkpointTimeout: checkpointTimeout,
		clock:             clock,
		finishedMap:       map[ShardId]bool{},
		finishedLck:       sync.Mutex{},
	}
}

func (m *metadataRepository) RegisterClient(ctx context.Context) (clientIndex int, totalClients int, err error) {
	namespace := m.getClientNamespace()
	_, err = m.repo.PutItem(ctx, m.repo.PutItemBuilder(), &ClientRecord{
		BaseRecord: BaseRecord{
			Namespace: namespace,
			Resource:  string(m.clientId),
			UpdatedAt: m.clock.Now(),
			Ttl:       mdl.Box(m.clock.Now().Add(m.clientTimeout).Unix()),
		},
	})
	if err != nil {
		return 0, 0, fmt.Errorf("failed to register client: %w", err)
	}

	qb := m.repo.QueryBuilder().WithHash(namespace).WithConsistentRead(true)
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

func (m *metadataRepository) IsShardFinished(ctx context.Context, shardId ShardId) (bool, error) {
	m.finishedLck.Lock()
	defer m.finishedLck.Unlock()

	if m.finishedMap[shardId] {
		return true, nil
	}

	namespace := m.getCheckpointNamespace()
	getQb := m.repo.GetItemBuilder().
		WithHash(namespace).
		WithRange(shardId)

	record := &CheckpointRecord{}
	getResult, err := m.repo.GetItem(ctx, getQb, record)
	if err != nil {
		return false, fmt.Errorf("failed to check if shard is finished: %w", err)
	}

	if getResult.IsFound && record.FinishedAt != nil {
		m.finishedMap[shardId] = true

		return true, nil
	}

	// we have either never yet consumed this shard, so it is not yet finished (as far as we know), or we know that it is
	// not yet finished

	return false, nil
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
				Ttl:       mdl.Box(m.clock.Now().Add(ShardTimeout).Unix()),
			},
			OwningClientId:    m.clientId,
			SequenceNumber:    "",
			LastShardIterator: "",
			FinishedAt:        nil,
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
		record.Ttl = mdl.Box(m.clock.Now().Add(ShardTimeout).Unix())
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
		shardIterator:       record.LastShardIterator,
		finalSequenceNumber: finalSequenceNumber,
		finishedAt:          record.FinishedAt,
	}, nil
}

func (m *metadataRepository) getClientNamespace() string {
	return fmt.Sprintf("client:%s:%s", m.appId.String(), m.stream)
}

func (m *metadataRepository) getCheckpointNamespace() string {
	return fmt.Sprintf("checkpoint:%s:%s", m.appId.String(), m.stream)
}
