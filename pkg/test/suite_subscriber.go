package test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/ddb"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"testing"
)

type SubscriberTestCase interface {
	GetName() string
	GetModelId() mdl.ModelId
	GetInput() interface{}
}

type subscriberTestCase struct {
	Name    string
	ModelId mdl.ModelId
	Input   interface{}
	Output  mdlsub.Model
}

func (s subscriberTestCase) GetName() string {
	return s.Name
}

func (s subscriberTestCase) GetModelId() mdl.ModelId {
	return s.ModelId
}

func (s subscriberTestCase) GetInput() interface{} {
	return s.Input
}

type TestingSuiteSubscriber interface {
	TestingSuite
	SetupTestCases() []SubscriberTestCase
	TestSubscriber(appUnderTest AppUnderTest, testCases []SubscriberTestCase)
}

func RunSubscriberTestSuite(t *testing.T, suite TestingSuiteSubscriber) {
	suite.SetT(t)

	RunTestCase(t, suite, func(appUnderTest AppUnderTest) {
		testCases := suite.SetupTestCases()
		suite.TestSubscriber(appUnderTest, testCases)
	})
}

type SubscriberTestSuite struct {
	Suite
}

func (s *SubscriberTestSuite) TestSubscriber(app AppUnderTest, testCases []SubscriberTestCase) {
	for _, sub := range testCases {
		s.publish(sub)
	}

	app.Stop()
	app.WaitDone()

	config := s.Env().Config()
	logger := s.Env().Logger()

	for _, sub := range testCases {
		configKey := mdlsub.GetSubscriberOutputConfigKey(sub.GetName())
		outputType := config.GetString(configKey)

		switch outputType {
		case mdlsub.OutputTypeDb:
			dbSub, ok := sub.(dbSubscriberTestCase)

			if !ok {
				s.FailNow("invalid subscription test case", "the test case for the subscription of %s has to be of the db type", sub.GetName())
				return
			}

			orm := db_repo.NewOrm(config, logger)
			fetcher := &DbSubscriberFetcher{
				t:    s.T(),
				orm:  orm,
				name: sub.GetName(),
			}

			dbSub.Assert(s.T(), fetcher)

		case mdlsub.OutputTypeDdb:
			ddbSub, ok := sub.(ddbSubscriberTestCase)

			if !ok {
				s.FailNow("invalid subscription test case", "the test case for the subscription of %s has to be of the ddb type", sub.GetName())
				return
			}

			fetcher := &DdbSubscriberFetcher{
				t: s.T(),
				repo: func(model interface{}) ddb.Repository {
					return ddb.NewRepository(config, logger, &ddb.Settings{
						ModelId: ddbSub.ModelIdTarget,
						Main: ddb.MainSettings{
							Model:              model,
							ReadCapacityUnits:  5,
							WriteCapacityUnits: 5,
						},
					})
				},
				name: sub.GetName(),
			}

			ddbSub.Assert(s.T(), fetcher)

		case mdlsub.OutputTypeKvstore:
			dbSub, ok := sub.(kvstoreSubscriberTestCase)

			if !ok {
				s.FailNow("invalid subscription test case", "the test case for the subscription of %s has to be of the kvstore type", sub.GetName())
				return
			}

			store := kvstore.NewConfigurableKvStore(config, logger, sub.GetName())
			fetcher := &KvstoreSubscriberFetcher{
				t:     s.T(),
				store: store,
				name:  sub.GetName(),
			}

			dbSub.Assert(s.T(), fetcher)
		}
	}
}

func (s SubscriberTestSuite) publish(sub SubscriberTestCase) {
	attrs := mdlsub.CreateMessageAttributes(sub.GetModelId(), "create", 0)

	streamName := fmt.Sprintf("subscriber-%s", sub.GetName())
	s.Env().StreamInput(streamName).Publish(sub.GetInput(), attrs)
}

func (s SubscriberTestSuite) DbTestCase(testCase DbSubscriberTestCase) SubscriberTestCase {
	modelId, err := mdl.ModelIdFromString(testCase.ModelId)

	if err != nil {
		s.FailNowf(err.Error(), "invalid modelId for subscription test case %s", testCase.Name)
	}

	return dbSubscriberTestCase{
		subscriberTestCase: subscriberTestCase{
			Name:    testCase.Name,
			ModelId: modelId,
			Input:   testCase.Input,
		},
		Assert: testCase.Assert,
	}
}

func (s SubscriberTestSuite) DdbTestCase(testCase DdbSubscriberTestCase) SubscriberTestCase {
	var err error
	var modelIdSource, modelIdTarget mdl.ModelId

	if modelIdSource, err = mdl.ModelIdFromString(testCase.SourceModelId); err != nil {
		s.FailNowf(err.Error(), "invalid source modelId for subscription test case %s", testCase.Name)
	}

	if modelIdTarget, err = mdl.ModelIdFromString(testCase.TargetModelId); err != nil {
		s.FailNowf(err.Error(), "invalid target modelId for subscription test case %s", testCase.Name)
	}

	return ddbSubscriberTestCase{
		subscriberTestCase: subscriberTestCase{
			Name:    testCase.Name,
			ModelId: modelIdSource,
			Input:   testCase.Input,
		},
		ModelIdTarget: modelIdTarget,
		Assert:        testCase.Assert,
	}
}

func (s SubscriberTestSuite) KvstoreTestCase(testCase KvstoreSubscriberTestCase) SubscriberTestCase {
	modelId, err := mdl.ModelIdFromString(testCase.ModelId)

	if err != nil {
		s.FailNowf(err.Error(), "invalid modelId for subscription test case %s", testCase.Name)
	}

	return kvstoreSubscriberTestCase{
		subscriberTestCase: subscriberTestCase{
			Name:    testCase.Name,
			ModelId: modelId,
			Input:   testCase.Input,
		},
		Assert: testCase.Assert,
	}
}

type DbSubscriberTestCase struct {
	Name    string
	ModelId string
	Input   interface{}
	Assert  DbSubscriberAssertion
}

type dbSubscriberTestCase struct {
	subscriberTestCase
	Assert DbSubscriberAssertion
}

type DbSubscriberAssertion func(t *testing.T, fetcher *DbSubscriberFetcher)

type DbSubscriberFetcher struct {
	t    *testing.T
	orm  *gorm.DB
	name string
}

func (f DbSubscriberFetcher) ByPrimaryKey(key interface{}, model interface{}) {
	res := f.orm.First(model, key)
	assert.NoErrorf(f.t, res.Error, "unexpected error on fetching db subscription %s", f.name)
}

type DdbSubscriberTestCase struct {
	Name          string
	SourceModelId string
	TargetModelId string
	Input         interface{}
	Assert        DdbSubscriberAssertion
}

type ddbSubscriberTestCase struct {
	subscriberTestCase
	ModelIdTarget mdl.ModelId
	Assert        DdbSubscriberAssertion
}

type DdbSubscriberAssertion func(t *testing.T, fetcher *DdbSubscriberFetcher)

type DdbSubscriberFetcher struct {
	t    *testing.T
	repo func(model interface{}) ddb.Repository
	name string
}

func (f DdbSubscriberFetcher) ByHash(hash interface{}, model interface{}) {
	repo := f.repo(model)
	qb := repo.GetItemBuilder().WithHash(hash)
	res, err := repo.GetItem(context.Background(), qb, model)

	assert.NoErrorf(f.t, err, "unexpected error on fetching db subscription %s", f.name)

	if err != nil {
		return
	}

	assert.True(f.t, res.IsFound)
}

func (f DdbSubscriberFetcher) ByHashAndRange(hash interface{}, rangeValue interface{}, model interface{}) {
	repo := f.repo(model)
	qb := repo.GetItemBuilder().WithHash(hash).WithRange(rangeValue)
	res, err := repo.GetItem(context.Background(), qb, model)

	assert.NoErrorf(f.t, err, "unexpected error on fetching db subscription %s", f.name)

	if err != nil {
		return
	}

	assert.True(f.t, res.IsFound)
}

type KvstoreSubscriberTestCase struct {
	Name    string
	ModelId string
	Input   interface{}
	Assert  KvStoreSubscriberAssertion
}

type kvstoreSubscriberTestCase struct {
	subscriberTestCase
	Assert KvStoreSubscriberAssertion
}

type KvStoreSubscriberAssertion func(t *testing.T, fetcher *KvstoreSubscriberFetcher)

type KvstoreSubscriberFetcher struct {
	t     *testing.T
	store kvstore.KvStore
	name  string
}

func (f KvstoreSubscriberFetcher) Get(key interface{}, model interface{}) {
	ok, err := f.store.Get(context.Background(), key, model)

	assert.Truef(f.t, ok, "model for subscription %s and key %v should be available in the store", f.name, key)
	assert.NoErrorf(f.t, err, "unexpected error on fetching kvstore subscription %s", f.name)
}
