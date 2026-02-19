package kinesis_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/ddb"
	ddbMocks "github.com/justtrackio/gosoline/pkg/ddb/mocks"
	logMocks "github.com/justtrackio/gosoline/pkg/log/mocks"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/test/matcher"
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type metadataRepositoryTestSuite struct {
	suite.Suite

	ctx                 context.Context
	logger              logMocks.LoggerMock
	stream              kinesis.Stream
	clientId            kinesis.ClientId
	clientNamespace     string
	shardId             kinesis.ShardId
	checkpointNamespace string
	repo                *ddbMocks.Repository
	settings            kinesis.Settings
	identity            cfg.Identity
	clock               clock.FakeClock
	metadataRepository  kinesis.MetadataRepository
}

func TestMetadataRepository(t *testing.T) {
	suite.Run(t, new(metadataRepositoryTestSuite))
}

func (s *metadataRepositoryTestSuite) SetupTest() {
	s.ctx = s.T().Context()
	s.logger = logMocks.NewLoggerMock(logMocks.WithTestingT(s.T()))
	s.stream = "testStream"
	s.clientId = kinesis.ClientId(uuid.New().NewV4())
	s.clientNamespace = string("client:gosoline-test-metadata-repository:testStream")
	s.shardId = kinesis.ShardId(uuid.New().NewV4())
	s.checkpointNamespace = string("checkpoint:gosoline-test-metadata-repository:testStream")
	s.repo = ddbMocks.NewRepository(s.T())
	s.settings = kinesis.Settings{
		DiscoverFrequency:        time.Minute * 10,
		CheckpointTimeoutPeriods: 5,
		PersistFrequency:         time.Second * 10,
		ClientExpirationPeriods:  5,
	}
	s.identity = cfg.Identity{
		Name: "test-suite",
		Env:  "test",
		Tags: cfg.Tags{
			"project": "gosoline",
			"family":  "metadata-repository",
		},
	}
	s.clock = clock.NewFakeClock()
	s.metadataRepository = kinesis.NewMetadataRepositoryWithInterfaces(s.logger, s.stream, s.clientId, s.repo, s.settings, "gosoline-test-metadata-repository", s.clock)
}

func (s *metadataRepositoryTestSuite) TestRegisterClient_PutItemError() {
	s.mockRegisterClientPutItem(fmt.Errorf("fail"))

	clientIndex, totalClients, err := s.metadataRepository.RegisterClient(s.ctx)
	s.Zero(clientIndex)
	s.Zero(totalClients)
	s.EqualError(err, "failed to register client: fail")
}

func (s *metadataRepositoryTestSuite) TestRegisterClient_QueryError() {
	s.mockRegisterClientPutItem(nil)
	s.mockRegisterClientQuery(0, fmt.Errorf("fail"))

	clientIndex, totalClients, err := s.metadataRepository.RegisterClient(s.ctx)
	s.Zero(clientIndex)
	s.Zero(totalClients)
	s.EqualError(err, "failed to list clients: fail")
}

func (s *metadataRepositoryTestSuite) TestRegisterClient_QueryInconsistent() {
	s.mockRegisterClientPutItem(nil)
	s.mockRegisterClientQuery(1, nil)

	clientIndex, totalClients, err := s.metadataRepository.RegisterClient(s.ctx)
	s.Zero(clientIndex)
	s.Zero(totalClients)
	s.EqualError(err, "failed to find client just written to ddb")
}

func (s *metadataRepositoryTestSuite) TestRegisterClient_Success() {
	s.mockRegisterClientPutItem(nil)
	s.mockRegisterClientQuery(5, nil)

	clientIndex, totalClients, err := s.metadataRepository.RegisterClient(s.ctx)
	s.Equal(2, clientIndex)
	s.Equal(5, totalClients)
	s.NoError(err)
}

func (s *metadataRepositoryTestSuite) mockRegisterClientPutItem(err error) {
	qb := ddbMocks.NewPutItemBuilder(s.T())

	s.repo.EXPECT().PutItemBuilder().Return(qb).Once()
	s.repo.EXPECT().PutItem(s.ctx, qb, &kinesis.ClientRecord{
		BaseRecord: kinesis.BaseRecord{
			Namespace: s.clientNamespace,
			Resource:  string(s.clientId),
			UpdatedAt: s.clock.Now(),
			Ttl:       mdl.Box(s.clock.Now().Add(time.Minute * 50).Unix()),
		},
	}).Return(&ddb.PutItemResult{}, err).Once()
}

func (s *metadataRepositoryTestSuite) mockRegisterClientQuery(resultCount int, err error) {
	qb := ddbMocks.NewQueryBuilder(s.T())
	qb.EXPECT().WithHash(s.clientNamespace).Return(qb).Once()
	qb.EXPECT().WithConsistentRead(true).Return(qb).Once()

	s.repo.EXPECT().QueryBuilder().Return(qb).Once()
	s.repo.EXPECT().Query(s.ctx, qb, &[]kinesis.ClientRecord{}).Run(func(ctx context.Context, qb ddb.QueryBuilder, result any) {
		clients := result.(*[]kinesis.ClientRecord)

		queryResult := make([]kinesis.ClientRecord, resultCount)
		for i := range queryResult {
			queryResult[i] = kinesis.ClientRecord{
				BaseRecord: kinesis.BaseRecord{
					Namespace: s.clientNamespace,
					Resource:  uuid.New().NewV4(),
					UpdatedAt: s.clock.Now(),
					Ttl:       mdl.Box(s.clock.Now().Add(time.Minute * 45).Unix()),
				},
			}
			if i == 2 {
				queryResult[i].Resource = string(s.clientId)
			}
		}

		*clients = queryResult
	}).Return(&ddb.QueryResult{}, err).Once()
}

func (s *metadataRepositoryTestSuite) TestDeregisterClient_DeleteError() {
	s.mockDeregisterClientDeleteItem(fmt.Errorf("fail"))

	err := s.metadataRepository.DeregisterClient(s.ctx)
	s.EqualError(err, "failed to deregister client: fail")
}

func (s *metadataRepositoryTestSuite) TestDeregisterClient_Success() {
	s.mockDeregisterClientDeleteItem(nil)

	err := s.metadataRepository.DeregisterClient(s.ctx)
	s.NoError(err)
}

func (s *metadataRepositoryTestSuite) mockDeregisterClientDeleteItem(err error) {
	qb := ddbMocks.NewDeleteItemBuilder(s.T())
	qb.EXPECT().WithHash(s.clientNamespace).Return(qb).Once()
	qb.EXPECT().WithRange(s.clientId).Return(qb).Once()

	s.repo.EXPECT().DeleteItemBuilder().Return(qb).Once()
	s.repo.EXPECT().DeleteItem(s.ctx, qb, &kinesis.ClientRecord{}).Return(&ddb.DeleteItemResult{}, err).Once()
}

func (s *metadataRepositoryTestSuite) TestIsShardFinished_ReadError() {
	s.mockIsShardFinished(false, nil, fmt.Errorf("fail"))

	finished, err := s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.EqualError(err, "failed to check if shard is finished: fail")
	s.False(finished)
}

func (s *metadataRepositoryTestSuite) TestIsShardFinished_RecordNotFound() {
	s.mockIsShardFinished(false, nil, nil)

	finished, err := s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.False(finished)
}

func (s *metadataRepositoryTestSuite) TestIsShardFinished_NotFinishedTwice() {
	s.mockIsShardFinished(true, nil, nil)
	s.mockIsShardFinished(true, nil, nil)

	finished, err := s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.False(finished)

	finished, err = s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.False(finished)
}

func (s *metadataRepositoryTestSuite) TestIsShardFinished_FirstReadThenCached() {
	s.mockIsShardFinished(true, mdl.Box(s.clock.Now()), nil)

	finished, err := s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.True(finished)

	finished, err = s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.True(finished)
}

func (s *metadataRepositoryTestSuite) mockIsShardFinished(found bool, finishedAt *time.Time, err error) {
	qb := ddbMocks.NewGetItemBuilder(s.T())
	qb.EXPECT().WithHash(s.checkpointNamespace).Return(qb).Once()
	qb.EXPECT().WithRange(s.shardId).Return(qb).Once()

	s.repo.EXPECT().GetItemBuilder().Return(qb).Once()
	s.repo.EXPECT().GetItem(s.ctx, qb, &kinesis.CheckpointRecord{}).Run(func(ctx context.Context, qb ddb.GetItemBuilder, result any) {
		record := result.(*kinesis.CheckpointRecord)

		if !found {
			return
		}

		*record = kinesis.CheckpointRecord{
			BaseRecord: kinesis.BaseRecord{
				Namespace: s.checkpointNamespace,
				Resource:  string(s.shardId),
			},
			FinishedAt: finishedAt,
		}
	}).Return(&ddb.GetItemResult{
		IsFound: found,
	}, err).Once()
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_ReadFail() {
	s.mockAcquireShardGetItem(false, "", 0, fmt.Errorf("fail"))

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.Nil(checkpoint)
	s.EqualError(err, "failed to read checkpoint record: fail")
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_StillOwned() {
	otherClientId := kinesis.ClientId(uuid.New().NewV4())

	s.mockAcquireShardGetItem(true, otherClientId, time.Second, nil)

	s.logger.EXPECT().Info(matcher.Context, "not trying to take over shard %s from %s, it is still in use", s.shardId, otherClientId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.Nil(checkpoint)
	s.NoError(err)
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_PutFail() {
	s.mockAcquireShardGetItem(false, "", 0, nil)
	s.mockAcquireShardPutItem("", "", false, fmt.Errorf("fail"))

	s.logger.EXPECT().Info(matcher.Context, "trying to use unused shard %s", s.shardId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.Nil(checkpoint)
	s.EqualError(err, "failed to write checkpoint record: fail")
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_PutConditionFail() {
	s.mockAcquireShardGetItem(false, "", 0, nil)
	s.mockAcquireShardPutItem("", "", true, nil)

	s.logger.EXPECT().Info(matcher.Context, "trying to use unused shard %s", s.shardId).Once()
	s.logger.EXPECT().Info(matcher.Context, "failed to acquire shard %s", s.shardId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.Nil(checkpoint)
	s.NoError(err)
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_SuccessNotFound() {
	s.mockAcquireShardGetItem(false, "", 0, nil)
	s.mockAcquireShardPutItem("", "", false, nil)

	s.logger.EXPECT().Info(matcher.Context, "trying to use unused shard %s", s.shardId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.NoError(err)
	s.NotNil(checkpoint)
	s.Equal(kinesis.SequenceNumber(""), checkpoint.GetSequenceNumber())

	s.testCheckpoint(checkpoint, "", "", false)
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_SuccessNotInUse() {
	s.mockAcquireShardGetItem(true, "", 0, nil)
	s.mockAcquireShardPutItem("1234", "1234==", false, nil)

	s.logger.EXPECT().Info(matcher.Context, "trying to take over shard %s from %s", s.shardId, kinesis.ClientId("nobody")).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.NoError(err)
	s.NotNil(checkpoint)
	s.Equal(kinesis.SequenceNumber("1234"), checkpoint.GetSequenceNumber())

	s.testCheckpoint(checkpoint, "1234", "1234==", true)
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_SuccessTakenOver() {
	otherClientId := kinesis.ClientId(uuid.New().NewV4())

	s.mockAcquireShardGetItem(true, otherClientId, time.Hour, nil)
	s.mockAcquireShardPutItem("1234", "1234==", false, nil)

	s.logger.EXPECT().Info(matcher.Context, "trying to take over shard %s from %s", s.shardId, otherClientId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.NoError(err)
	s.NotNil(checkpoint)
	s.Equal(kinesis.SequenceNumber("1234"), checkpoint.GetSequenceNumber())

	s.testCheckpoint(checkpoint, "1234", "1234==", false)
}

func (s *metadataRepositoryTestSuite) mockAcquireShardGetItem(found bool, owningClientId kinesis.ClientId, age time.Duration, err error) {
	qb := ddbMocks.NewGetItemBuilder(s.T())
	qb.EXPECT().WithHash(s.checkpointNamespace).Return(qb).Once()
	qb.EXPECT().WithRange(s.shardId).Return(qb).Once()
	qb.EXPECT().WithConsistentRead(true).Return(qb).Once()

	s.repo.EXPECT().GetItemBuilder().Return(qb).Once()
	s.repo.EXPECT().GetItem(s.ctx, qb, &kinesis.CheckpointRecord{}).Run(func(ctx context.Context, qb ddb.GetItemBuilder, result any) {
		record := result.(*kinesis.CheckpointRecord)

		if !found {
			return
		}

		*record = kinesis.CheckpointRecord{
			BaseRecord: kinesis.BaseRecord{
				Namespace: s.checkpointNamespace,
				Resource:  string(s.shardId),
				UpdatedAt: s.clock.Now().Add(-age),
				Ttl:       mdl.Box(s.clock.Now().Add(kinesis.ShardTimeout - age).Unix()),
			},
			OwningClientId:    owningClientId,
			SequenceNumber:    "1234",
			LastShardIterator: "1234==",
			FinishedAt:        nil,
		}
	}).Return(&ddb.GetItemResult{
		IsFound: found,
	}, err).Once()
}

func (s *metadataRepositoryTestSuite) mockAcquireShardPutItem(sequenceNumber kinesis.SequenceNumber, shardIterator kinesis.ShardIterator, conditionalCheckFailed bool, err error) {
	qb := ddbMocks.NewPutItemBuilder(s.T())
	qb.EXPECT().WithCondition(ddb.AttributeNotExists("owningClientId").Or(ddb.Lte("updatedAt", s.clock.Now().Add(-time.Minute)))).Return(qb).Once()

	s.repo.EXPECT().PutItemBuilder().Return(qb).Once()
	s.repo.EXPECT().PutItem(matcher.Context, qb, &kinesis.CheckpointRecord{
		BaseRecord: kinesis.BaseRecord{
			Namespace: s.checkpointNamespace,
			Resource:  string(s.shardId),
			UpdatedAt: s.clock.Now(),
			Ttl:       mdl.Box(s.clock.Now().Add(kinesis.ShardTimeout).Unix()),
		},
		OwningClientId:    s.clientId,
		SequenceNumber:    sequenceNumber,
		LastShardIterator: shardIterator,
		FinishedAt:        nil,
	}).Return(&ddb.PutItemResult{
		ConditionalCheckFailed: conditionalCheckFailed,
	}, err).Once()
}

func (s *metadataRepositoryTestSuite) testCheckpoint(checkpoint kinesis.Checkpoint, initialSequenceNumber kinesis.SequenceNumber, lastShardIterator kinesis.ShardIterator, wasAlreadyReleased bool) {
	s.Run("Persist_NotDirty", func() {
		s.mockCheckpointPersist(nil, initialSequenceNumber, lastShardIterator, false, nil)

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.NoError(err)
	})

	s.Run("Advance_Success", func() {
		err := checkpoint.Advance("2000", "2000==")
		s.NoError(err)
	})

	s.Run("Persist_Error", func() {
		s.mockCheckpointPersist(nil, "2000", "2000==", false, fmt.Errorf("fail"))

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.EqualError(err, "failed to persist checkpoint: fail")
	})

	s.Run("Persist_ConditionalCheckFailed", func() {
		s.mockCheckpointPersist(nil, "2000", "2000==", true, nil)

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.Equal(err, kinesis.ErrCheckpointNoLongerOwned)
	})

	s.Run("Persist_Success", func() {
		s.mockCheckpointPersist(nil, "2000", "2000==", false, nil)

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.NoError(err)
	})

	s.Run("Persist_NoLongerDirty", func() {
		s.mockCheckpointPersist(nil, "2000", "2000==", false, nil)

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.NoError(err)
	})

	s.Run("Done_Success", func() {
		err := checkpoint.Done("2000")
		s.NoError(err)
	})

	// we marked it as done, so we need to record the time we finished it
	finishedAt := mdl.Box(s.clock.Now())

	s.Run("PersistFinished_Error", func() {
		s.mockCheckpointPersist(finishedAt, "2000", "2000==", false, fmt.Errorf("fail"))

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.EqualError(err, "failed to persist checkpoint: fail")
	})

	s.Run("PersistFinished_ConditionalCheckFailed", func() {
		s.mockCheckpointPersist(finishedAt, "2000", "2000==", true, nil)

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.Equal(err, kinesis.ErrCheckpointNoLongerOwned)
	})

	s.Run("PersistFinished_Success", func() {
		s.mockCheckpointPersist(finishedAt, "2000", "2000==", false, nil)

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.True(shouldRelease)
		s.NoError(err)
	})

	s.Run("Release_Error", func() {
		s.mockCheckpointRelease("2000", "2000==", false, fmt.Errorf("fail"))

		err := checkpoint.Release(s.ctx)
		s.EqualError(err, "failed to release checkpoint: fail")
	})

	// we can either fail to release it or release it, but not both
	if wasAlreadyReleased {
		s.Run("Release_AlreadyReleased", func() {
			s.mockCheckpointRelease("2000", "2000==", true, nil)

			err := checkpoint.Release(s.ctx)
			s.Equal(err, kinesis.ErrCheckpointAlreadyReleased)
		})
	} else {
		s.Run("Release_Success", func() {
			s.mockCheckpointRelease("2000", "2000==", false, nil)

			err := checkpoint.Release(s.ctx)
			s.NoError(err)
		})
	}

	s.Run("Advance_AlreadyReleased", func() {
		err := checkpoint.Advance("3000", "3000==")
		s.EqualError(err, "can not advance already released checkpoint: "+conc.ErrAlreadyPoisoned.Error())
	})

	s.Run("Done_AlreadyReleased", func() {
		err := checkpoint.Done("3000")
		s.EqualError(err, "can not mark already released checkpoint as done: "+conc.ErrAlreadyPoisoned.Error())
	})

	s.Run("Persist_AlreadyReleased", func() {
		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.EqualError(err, "can not persist already released checkpoint: "+conc.ErrAlreadyPoisoned.Error())
	})
}

func (s *metadataRepositoryTestSuite) mockCheckpointPersist(finishedAt *time.Time, sequenceNumber kinesis.SequenceNumber, shardIterator kinesis.ShardIterator, conditionalCheckFailed bool, err error) {
	// let the time run on to not always get the same numbers
	s.clock.Advance(time.Second)

	qb := ddbMocks.NewPutItemBuilder(s.T())
	qb.EXPECT().WithCondition(ddb.Eq("owningClientId", s.clientId)).Return(qb).Once()

	s.repo.EXPECT().PutItemBuilder().Return(qb).Once()
	s.repo.EXPECT().PutItem(s.ctx, qb, &kinesis.CheckpointRecord{
		BaseRecord: kinesis.BaseRecord{
			Namespace: s.checkpointNamespace,
			Resource:  string(s.shardId),
			UpdatedAt: s.clock.Now(),
			Ttl:       mdl.Box(s.clock.Now().Add(kinesis.ShardTimeout).Unix()),
		},
		OwningClientId:    s.clientId,
		SequenceNumber:    sequenceNumber,
		LastShardIterator: shardIterator,
		FinishedAt:        finishedAt,
	}).Return(&ddb.PutItemResult{
		ConditionalCheckFailed: conditionalCheckFailed,
	}, err).Once()
}

func (s *metadataRepositoryTestSuite) mockCheckpointRelease(sequenceNumber kinesis.SequenceNumber, shardIterator kinesis.ShardIterator, conditionalCheckFailed bool, err error) {
	// let the time run on to not always get the same numbers
	s.clock.Advance(time.Second)

	qb := ddbMocks.NewUpdateItemBuilder(s.T())
	qb.EXPECT().WithHash(s.checkpointNamespace).Return(qb).Once()
	qb.EXPECT().WithRange(s.shardId).Return(qb).Once()
	qb.EXPECT().Remove("owningClientId").Return(qb).Once()
	qb.EXPECT().Set("updatedAt", s.clock.Now()).Return(qb).Once()
	qb.EXPECT().Set("ttl", mdl.Box(s.clock.Now().Add(kinesis.ShardTimeout).Unix())).Return(qb).Once()
	qb.EXPECT().Set("sequenceNumber", sequenceNumber).Return(qb).Once()
	qb.EXPECT().Set("lastShardIterator", shardIterator).Return(qb).Once()
	qb.EXPECT().WithCondition(ddb.Eq("owningClientId", s.clientId)).Return(qb).Once()

	s.repo.EXPECT().UpdateItemBuilder().Return(qb).Once()
	s.repo.EXPECT().UpdateItem(s.ctx, qb, &kinesis.CheckpointRecord{}).Return(&ddb.UpdateItemResult{
		ConditionalCheckFailed: conditionalCheckFailed,
	}, err).Once()
}
