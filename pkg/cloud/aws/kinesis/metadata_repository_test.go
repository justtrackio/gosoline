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
	"github.com/justtrackio/gosoline/pkg/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type metadataRepositoryTestSuite struct {
	suite.Suite

	ctx                 context.Context
	logger              *logMocks.Logger
	stream              kinesis.Stream
	clientId            kinesis.ClientId
	clientNamespace     string
	shardId             kinesis.ShardId
	checkpointNamespace string
	repo                *ddbMocks.Repository
	settings            kinesis.Settings
	clock               clock.FakeClock
	metadataRepository  kinesis.MetadataRepository
}

func TestMetadataRepository(t *testing.T) {
	suite.Run(t, new(metadataRepositoryTestSuite))
}

func (s *metadataRepositoryTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.logger = new(logMocks.Logger)
	s.stream = "testStream"
	s.clientId = kinesis.ClientId(uuid.New().NewV4())
	s.clientNamespace = string("client:gosoline-test-metadata-repository-test-suite:" + s.stream)
	s.shardId = kinesis.ShardId(uuid.New().NewV4())
	s.checkpointNamespace = string("checkpoint:gosoline-test-metadata-repository-test-suite:" + s.stream)
	s.repo = new(ddbMocks.Repository)
	s.settings = kinesis.Settings{
		AppId: cfg.AppId{
			Project:     "gosoline",
			Environment: "test",
			Family:      "metadata-repository",
			Application: "test-suite",
		},
		DiscoverFrequency: time.Minute * 10,
		PersistFrequency:  time.Second * 10,
	}
	s.clock = clock.NewFakeClock()

	s.metadataRepository = kinesis.NewMetadataRepositoryWithInterfaces(s.logger, s.stream, s.clientId, s.repo, s.settings, s.clock)
}

func (s *metadataRepositoryTestSuite) TearDownTest() {
	s.logger.AssertExpectations(s.T())
	s.repo.AssertExpectations(s.T())
}

func (s *metadataRepositoryTestSuite) TestRegisterClient_PutItemError() {
	putQb := s.mockRegisterClientPutItem(fmt.Errorf("fail"))
	defer putQb.AssertExpectations(s.T())

	clientIndex, totalClients, err := s.metadataRepository.RegisterClient(s.ctx)
	s.Zero(clientIndex)
	s.Zero(totalClients)
	s.EqualError(err, "failed to register client: fail")
}

func (s *metadataRepositoryTestSuite) TestRegisterClient_QueryError() {
	putQb := s.mockRegisterClientPutItem(nil)
	defer putQb.AssertExpectations(s.T())

	queryQb := s.mockRegisterClientQuery(0, fmt.Errorf("fail"))
	defer queryQb.AssertExpectations(s.T())

	clientIndex, totalClients, err := s.metadataRepository.RegisterClient(s.ctx)
	s.Zero(clientIndex)
	s.Zero(totalClients)
	s.EqualError(err, "failed to list clients: fail")
}

func (s *metadataRepositoryTestSuite) TestRegisterClient_QueryInconsistent() {
	putQb := s.mockRegisterClientPutItem(nil)
	defer putQb.AssertExpectations(s.T())

	queryQb := s.mockRegisterClientQuery(1, nil)
	defer queryQb.AssertExpectations(s.T())

	clientIndex, totalClients, err := s.metadataRepository.RegisterClient(s.ctx)
	s.Zero(clientIndex)
	s.Zero(totalClients)
	s.EqualError(err, "failed to find client just written to ddb")
}

func (s *metadataRepositoryTestSuite) TestRegisterClient_Success() {
	putQb := s.mockRegisterClientPutItem(nil)
	defer putQb.AssertExpectations(s.T())

	queryQb := s.mockRegisterClientQuery(5, nil)
	defer queryQb.AssertExpectations(s.T())

	clientIndex, totalClients, err := s.metadataRepository.RegisterClient(s.ctx)
	s.Equal(2, clientIndex)
	s.Equal(5, totalClients)
	s.NoError(err)
}

func (s *metadataRepositoryTestSuite) mockRegisterClientPutItem(err error) *ddbMocks.PutItemBuilder {
	qb := new(ddbMocks.PutItemBuilder)

	s.repo.On("PutItemBuilder").Return(qb).Once()
	s.repo.On("PutItem", s.ctx, qb, &kinesis.ClientRecord{
		BaseRecord: kinesis.BaseRecord{
			Namespace: s.clientNamespace,
			Resource:  string(s.clientId),
			UpdatedAt: s.clock.Now(),
			Ttl:       mdl.Int64(s.clock.Now().Add(time.Minute * 50).Unix()),
		},
	}).Return(&ddb.PutItemResult{}, err).Once()

	return qb
}

func (s *metadataRepositoryTestSuite) mockRegisterClientQuery(resultCount int, err error) *ddbMocks.QueryBuilder {
	qb := new(ddbMocks.QueryBuilder)
	qb.On("WithHash", s.clientNamespace).Return(qb).Once()

	s.repo.On("QueryBuilder").Return(qb).Once()
	s.repo.On("Query", s.ctx, qb, &[]kinesis.ClientRecord{}).Run(func(args mock.Arguments) {
		clients := args.Get(2).(*[]kinesis.ClientRecord)

		queryResult := make([]kinesis.ClientRecord, resultCount)
		for i := range queryResult {
			queryResult[i] = kinesis.ClientRecord{
				BaseRecord: kinesis.BaseRecord{
					Namespace: s.clientNamespace,
					Resource:  uuid.New().NewV4(),
					UpdatedAt: s.clock.Now(),
					Ttl:       mdl.Int64(s.clock.Now().Add(time.Minute * 45).Unix()),
				},
			}
			if i == 2 {
				queryResult[i].Resource = string(s.clientId)
			}
		}

		*clients = queryResult
	}).Return(&ddb.QueryResult{}, err).Once()

	return qb
}

func (s *metadataRepositoryTestSuite) TestDeregisterClient_DeleteError() {
	qb := s.mockDeregisterClientDeleteItem(fmt.Errorf("fail"))
	defer qb.AssertExpectations(s.T())

	err := s.metadataRepository.DeregisterClient(s.ctx)
	s.EqualError(err, "failed to deregister client: fail")
}

func (s *metadataRepositoryTestSuite) TestDeregisterClient_Success() {
	qb := s.mockDeregisterClientDeleteItem(nil)
	defer qb.AssertExpectations(s.T())

	err := s.metadataRepository.DeregisterClient(s.ctx)
	s.NoError(err)
}

func (s *metadataRepositoryTestSuite) mockDeregisterClientDeleteItem(err error) *ddbMocks.DeleteItemBuilder {
	qb := new(ddbMocks.DeleteItemBuilder)
	qb.On("WithHash", s.clientNamespace).Return(qb).Once()
	qb.On("WithRange", s.clientId).Return(qb).Once()

	s.repo.On("DeleteItemBuilder").Return(qb).Once()
	s.repo.On("DeleteItem", s.ctx, qb, &kinesis.ClientRecord{}).Return(&ddb.DeleteItemResult{}, err).Once()

	return qb
}

func (s *metadataRepositoryTestSuite) TestIsShardFinished_ReadError() {
	qb := s.mockIsShardFinished(false, nil, fmt.Errorf("fail"))
	defer qb.AssertExpectations(s.T())

	finished, err := s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.EqualError(err, "failed to check if shard is finished: fail")
	s.False(finished)
}

func (s *metadataRepositoryTestSuite) TestIsShardFinished_RecordNotFound() {
	qb := s.mockIsShardFinished(false, nil, nil)
	defer qb.AssertExpectations(s.T())

	finished, err := s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.False(finished)
}

func (s *metadataRepositoryTestSuite) TestIsShardFinished_NotFinishedTwice() {
	qb1 := s.mockIsShardFinished(true, nil, nil)
	defer qb1.AssertExpectations(s.T())
	qb2 := s.mockIsShardFinished(true, nil, nil)
	defer qb2.AssertExpectations(s.T())

	finished, err := s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.False(finished)

	finished, err = s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.False(finished)
}

func (s *metadataRepositoryTestSuite) TestIsShardFinished_FirstReadThenCached() {
	qb := s.mockIsShardFinished(true, mdl.Time(s.clock.Now()), nil)
	defer qb.AssertExpectations(s.T())

	finished, err := s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.True(finished)

	finished, err = s.metadataRepository.IsShardFinished(s.ctx, s.shardId)
	s.NoError(err)
	s.True(finished)
}

func (s *metadataRepositoryTestSuite) mockIsShardFinished(found bool, finishedAt *time.Time, err error) *ddbMocks.GetItemBuilder {
	qb := new(ddbMocks.GetItemBuilder)
	qb.On("WithHash", s.checkpointNamespace).Return(qb).Once()
	qb.On("WithRange", s.shardId).Return(qb).Once()

	s.repo.On("GetItemBuilder").Return(qb).Once()
	s.repo.On("GetItem", s.ctx, qb, &kinesis.CheckpointRecord{}).Run(func(args mock.Arguments) {
		record := args.Get(2).(*kinesis.CheckpointRecord)

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

	return qb
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_ReadFail() {
	qb := s.mockAcquireShardGetItem(false, "", 0, fmt.Errorf("fail"))
	defer qb.AssertExpectations(s.T())

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.Nil(checkpoint)
	s.EqualError(err, "failed to read checkpoint record: fail")
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_StillOwned() {
	otherClientId := kinesis.ClientId(uuid.New().NewV4())

	qb := s.mockAcquireShardGetItem(true, otherClientId, time.Second, nil)
	defer qb.AssertExpectations(s.T())

	s.logger.On("Info", "not trying to take over shard %s from %s, it is still in use", s.shardId, otherClientId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.Nil(checkpoint)
	s.NoError(err)
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_PutFail() {
	getQb := s.mockAcquireShardGetItem(false, "", 0, nil)
	defer getQb.AssertExpectations(s.T())

	putQb := s.mockAcquireShardPutItem("", false, fmt.Errorf("fail"))
	defer putQb.AssertExpectations(s.T())

	s.logger.On("Info", "trying to use unused shard %s", s.shardId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.Nil(checkpoint)
	s.EqualError(err, "failed to write checkpoint record: fail")
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_PutConditionFail() {
	getQb := s.mockAcquireShardGetItem(false, "", 0, nil)
	defer getQb.AssertExpectations(s.T())

	putQb := s.mockAcquireShardPutItem("", true, nil)
	defer putQb.AssertExpectations(s.T())

	s.logger.On("Info", "trying to use unused shard %s", s.shardId).Once()
	s.logger.On("Info", "failed to acquire shard %s", s.shardId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.Nil(checkpoint)
	s.NoError(err)
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_SuccessNotFound() {
	getQb := s.mockAcquireShardGetItem(false, "", 0, nil)
	defer getQb.AssertExpectations(s.T())

	putQb := s.mockAcquireShardPutItem("", false, nil)
	defer putQb.AssertExpectations(s.T())

	s.logger.On("Info", "trying to use unused shard %s", s.shardId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.NotNil(checkpoint)
	s.Equal(kinesis.SequenceNumber(""), checkpoint.GetSequenceNumber())
	s.NoError(err)

	s.testCheckpoint(checkpoint, "", false)
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_SuccessNotInUse() {
	getQb := s.mockAcquireShardGetItem(true, "", 0, nil)
	defer getQb.AssertExpectations(s.T())

	putQb := s.mockAcquireShardPutItem("1234", false, nil)
	defer putQb.AssertExpectations(s.T())

	s.logger.On("Info", "trying to take over shard %s from %s", s.shardId, kinesis.ClientId("nobody")).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.NotNil(checkpoint)
	s.Equal(kinesis.SequenceNumber("1234"), checkpoint.GetSequenceNumber())
	s.NoError(err)

	s.testCheckpoint(checkpoint, "1234", true)
}

func (s *metadataRepositoryTestSuite) TestAcquireShard_SuccessTakenOver() {
	otherClientId := kinesis.ClientId(uuid.New().NewV4())

	getQb := s.mockAcquireShardGetItem(true, otherClientId, time.Hour, nil)
	defer getQb.AssertExpectations(s.T())

	putQb := s.mockAcquireShardPutItem("1234", false, nil)
	defer putQb.AssertExpectations(s.T())

	s.logger.On("Info", "trying to take over shard %s from %s", s.shardId, otherClientId).Once()

	checkpoint, err := s.metadataRepository.AcquireShard(s.ctx, s.shardId)
	s.NotNil(checkpoint)
	s.Equal(kinesis.SequenceNumber("1234"), checkpoint.GetSequenceNumber())
	s.NoError(err)

	s.testCheckpoint(checkpoint, "1234", false)
}

func (s *metadataRepositoryTestSuite) mockAcquireShardGetItem(found bool, owningClientId kinesis.ClientId, age time.Duration, err error) *ddbMocks.GetItemBuilder {
	qb := new(ddbMocks.GetItemBuilder)
	qb.On("WithHash", s.checkpointNamespace).Return(qb).Once()
	qb.On("WithRange", s.shardId).Return(qb).Once()
	qb.On("WithConsistentRead", true).Return(qb).Once()

	s.repo.On("GetItemBuilder").Return(qb).Once()
	s.repo.On("GetItem", s.ctx, qb, &kinesis.CheckpointRecord{}).Run(func(args mock.Arguments) {
		record := args.Get(2).(*kinesis.CheckpointRecord)

		if !found {
			return
		}

		*record = kinesis.CheckpointRecord{
			BaseRecord: kinesis.BaseRecord{
				Namespace: s.checkpointNamespace,
				Resource:  string(s.shardId),
				UpdatedAt: s.clock.Now().Add(-age),
				Ttl:       mdl.Int64(s.clock.Now().Add(kinesis.ShardTimeout - age).Unix()),
			},
			OwningClientId: owningClientId,
			SequenceNumber: "1234",
			FinishedAt:     nil,
		}
	}).Return(&ddb.GetItemResult{
		IsFound: found,
	}, err).Once()

	return qb
}

func (s *metadataRepositoryTestSuite) mockAcquireShardPutItem(sequenceNumber kinesis.SequenceNumber, conditionalCheckFailed bool, err error) *ddbMocks.PutItemBuilder {
	qb := new(ddbMocks.PutItemBuilder)
	qb.On("WithCondition", ddb.AttributeNotExists("owningClientId").Or(ddb.Lte("updatedAt", s.clock.Now().Add(-time.Minute)))).Return(qb).Once()

	s.repo.On("PutItemBuilder").Return(qb).Once()
	s.repo.On("PutItem", s.ctx, qb, &kinesis.CheckpointRecord{
		BaseRecord: kinesis.BaseRecord{
			Namespace: s.checkpointNamespace,
			Resource:  string(s.shardId),
			UpdatedAt: s.clock.Now(),
			Ttl:       mdl.Int64(s.clock.Now().Add(kinesis.ShardTimeout).Unix()),
		},
		OwningClientId: s.clientId,
		SequenceNumber: sequenceNumber,
		FinishedAt:     nil,
	}).Return(&ddb.PutItemResult{
		ConditionalCheckFailed: conditionalCheckFailed,
	}, err).Once()

	return qb
}

func (s *metadataRepositoryTestSuite) testCheckpoint(checkpoint kinesis.Checkpoint, initialSequenceNumber kinesis.SequenceNumber, wasAlreadyReleased bool) {
	s.Run("Persist_NotDirty", func() {
		qb := s.mockCheckpointPersist(nil, initialSequenceNumber, false, nil)
		defer qb.AssertExpectations(s.T())

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.NoError(err)
	})

	s.Run("Advance_Success", func() {
		err := checkpoint.Advance("2000")
		s.NoError(err)
	})

	s.Run("Persist_Error", func() {
		qb := s.mockCheckpointPersist(nil, "2000", false, fmt.Errorf("fail"))
		defer qb.AssertExpectations(s.T())

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.EqualError(err, "failed to persist checkpoint: fail")
	})

	s.Run("Persist_ConditionalCheckFailed", func() {
		qb := s.mockCheckpointPersist(nil, "2000", true, nil)
		defer qb.AssertExpectations(s.T())

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.Equal(err, kinesis.ErrCheckpointNoLongerOwned)
	})

	s.Run("Persist_Success", func() {
		qb := s.mockCheckpointPersist(nil, "2000", false, nil)
		defer qb.AssertExpectations(s.T())

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.NoError(err)
	})

	s.Run("Persist_NoLongerDirty", func() {
		qb := s.mockCheckpointPersist(nil, "2000", false, nil)
		defer qb.AssertExpectations(s.T())

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.NoError(err)
	})

	s.Run("Done_Success", func() {
		err := checkpoint.Done("2000")
		s.NoError(err)
	})

	// we marked it as done, so we need to record the time we finished it
	finishedAt := mdl.Time(s.clock.Now())

	s.Run("PersistFinished_Error", func() {
		qb := s.mockCheckpointPersist(finishedAt, "2000", false, fmt.Errorf("fail"))
		defer qb.AssertExpectations(s.T())

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.EqualError(err, "failed to persist checkpoint: fail")
	})

	s.Run("PersistFinished_ConditionalCheckFailed", func() {
		qb := s.mockCheckpointPersist(finishedAt, "2000", true, nil)
		defer qb.AssertExpectations(s.T())

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.False(shouldRelease)
		s.Equal(err, kinesis.ErrCheckpointNoLongerOwned)
	})

	s.Run("PersistFinished_Success", func() {
		qb := s.mockCheckpointPersist(finishedAt, "2000", false, nil)
		defer qb.AssertExpectations(s.T())

		shouldRelease, err := checkpoint.Persist(s.ctx)
		s.True(shouldRelease)
		s.NoError(err)
	})

	s.Run("Release_Error", func() {
		qb := s.mockCheckpointRelease("2000", false, fmt.Errorf("fail"))
		defer qb.AssertExpectations(s.T())

		err := checkpoint.Release(s.ctx)
		s.EqualError(err, "failed to release checkpoint: fail")
	})

	// we can either fail to release it or release it, but not both
	if wasAlreadyReleased {
		s.Run("Release_AlreadyReleased", func() {
			qb := s.mockCheckpointRelease("2000", true, nil)
			defer qb.AssertExpectations(s.T())

			err := checkpoint.Release(s.ctx)
			s.Equal(err, kinesis.ErrCheckpointAlreadyReleased)
		})
	} else {
		s.Run("Release_Success", func() {
			qb := s.mockCheckpointRelease("2000", false, nil)
			defer qb.AssertExpectations(s.T())

			err := checkpoint.Release(s.ctx)
			s.NoError(err)
		})
	}

	s.Run("Advance_AlreadyReleased", func() {
		err := checkpoint.Advance("3000")
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

func (s *metadataRepositoryTestSuite) mockCheckpointPersist(finishedAt *time.Time, sequenceNumber kinesis.SequenceNumber, conditionalCheckFailed bool, err error) *ddbMocks.PutItemBuilder {
	// let the time run on to not always get the same numbers
	s.clock.Advance(time.Second)

	qb := new(ddbMocks.PutItemBuilder)
	qb.On("WithCondition", ddb.Eq("owningClientId", s.clientId)).Return(qb).Once()

	s.repo.On("PutItemBuilder").Return(qb).Once()
	s.repo.On("PutItem", s.ctx, qb, &kinesis.CheckpointRecord{
		BaseRecord: kinesis.BaseRecord{
			Namespace: s.checkpointNamespace,
			Resource:  string(s.shardId),
			UpdatedAt: s.clock.Now(),
			Ttl:       mdl.Int64(s.clock.Now().Add(kinesis.ShardTimeout).Unix()),
		},
		OwningClientId: s.clientId,
		SequenceNumber: sequenceNumber,
		FinishedAt:     finishedAt,
	}).Return(&ddb.PutItemResult{
		ConditionalCheckFailed: conditionalCheckFailed,
	}, err).Once()

	return qb
}

func (s *metadataRepositoryTestSuite) mockCheckpointRelease(sequenceNumber kinesis.SequenceNumber, conditionalCheckFailed bool, err error) *ddbMocks.UpdateItemBuilder {
	// let the time run on to not always get the same numbers
	s.clock.Advance(time.Second)

	qb := new(ddbMocks.UpdateItemBuilder)
	qb.On("WithHash", s.checkpointNamespace).Return(qb).Once()
	qb.On("WithRange", s.shardId).Return(qb).Once()
	qb.On("Remove", "owningClientId").Return(qb).Once()
	qb.On("Set", "updatedAt", s.clock.Now()).Return(qb).Once()
	qb.On("Set", "ttl", mdl.Int64(s.clock.Now().Add(kinesis.ShardTimeout).Unix())).Return(qb).Once()
	qb.On("Set", "sequenceNumber", sequenceNumber).Return(qb).Once()
	qb.On("WithCondition", ddb.Eq("owningClientId", s.clientId)).Return(qb).Once()

	s.repo.On("UpdateItemBuilder").Return(qb).Once()
	s.repo.On("UpdateItem", s.ctx, qb, &kinesis.CheckpointRecord{}).Return(&ddb.UpdateItemResult{
		ConditionalCheckFailed: conditionalCheckFailed,
	}, err).Once()

	return qb
}
