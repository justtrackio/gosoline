package test

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/db-repo"
	"github.com/applike/gosoline/pkg/kvstore"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/jinzhu/gorm"
	"github.com/stretchr/testify/assert"
	"testing"
)

type SubscriptionTestCase interface {
	GetName() string
	GetModelId() mdl.ModelId
	GetInput() interface{}
}

type subscriptionTestCase struct {
	Name    string
	ModelId mdl.ModelId
	Input   interface{}
	Output  mdlsub.Model
}

func (s subscriptionTestCase) GetName() string {
	return s.Name
}

func (s subscriptionTestCase) GetModelId() mdl.ModelId {
	return s.ModelId
}

func (s subscriptionTestCase) GetInput() interface{} {
	return s.Input
}

type TestingSuiteSubscriber interface {
	TestingSuite
	SetupSubscriptions() []SubscriptionTestCase
	TestSubscriptions(appUnderTest AppUnderTest, subscriptions []SubscriptionTestCase)
}

func RunSubscriptionTestSuite(t *testing.T, suite TestingSuiteSubscriber) {
	suite.SetT(t)

	RunTestCase(t, suite, func(appUnderTest AppUnderTest) {
		subscriptions := suite.SetupSubscriptions()
		suite.TestSubscriptions(appUnderTest, subscriptions)
	})
}

type SubscriberTestSuite struct {
	Suite
}

func (s *SubscriberTestSuite) TestSubscriptions(app AppUnderTest, subscriptions []SubscriptionTestCase) {
	for _, sub := range subscriptions {
		s.publish(sub)
	}

	app.Stop()
	app.WaitDone()

	config := s.Env().Config()
	logger := s.Env().Logger()

	for _, sub := range subscriptions {
		configKey := mdlsub.GetSubscriberOutputConfigKey(sub.GetName())
		outputType := config.GetString(configKey)

		switch outputType {
		case mdlsub.OutputTypeDb:
			dbSub, ok := sub.(dbSubscriptionTestCase)

			if !ok {
				s.FailNow("invalid subscription test case", "the test case for the subscription of %s has to be of the db type", sub.GetName())
				return
			}

			orm := db_repo.NewOrm(config, logger)
			fetcher := &DbSubscriptionFetcher{
				t:    s.T(),
				orm:  orm,
				name: sub.GetName(),
			}

			dbSub.Assert(s.T(), fetcher)

		case mdlsub.OutputTypeKvstore:
			dbSub, ok := sub.(kvstoreSubscriptionTestCase)

			if !ok {
				s.FailNow("invalid subscription test case", "the test case for the subscription of %s has to be of the kvstore type", sub.GetName())
				return
			}

			store := kvstore.NewConfigurableKvStore(config, logger, sub.GetName())
			fetcher := &KvstoreSubscriptionFetcher{
				t:     s.T(),
				store: store,
				name:  sub.GetName(),
			}

			dbSub.Assert(s.T(), fetcher)
		}
	}
}

func (s SubscriberTestSuite) publish(subscription SubscriptionTestCase) {
	attrs := mdlsub.CreateMessageAttributes(subscription.GetModelId(), "create", 0)

	streamName := fmt.Sprintf("subscriber-%s", subscription.GetName())
	s.Env().StreamInput(streamName).Publish(subscription.GetInput(), attrs)
}

func (s SubscriberTestSuite) DbTestCase(name string, modelIdStr string, input interface{}, assert DbSubscriptionAssertion) SubscriptionTestCase {
	modelId, err := mdl.ModelIdFromString(modelIdStr)

	if err != nil {
		s.FailNowf(err.Error(), "invalid modelId for subscription test case %s", name)
	}

	return dbSubscriptionTestCase{
		subscriptionTestCase: subscriptionTestCase{
			Name:    name,
			ModelId: modelId,
			Input:   input,
		},
		Assert: assert,
	}
}

func (s SubscriberTestSuite) KvstoreTestCase(name string, modelIdStr string, input interface{}, assert KvStoreSubscriptionAssertion) SubscriptionTestCase {
	modelId, err := mdl.ModelIdFromString(modelIdStr)

	if err != nil {
		s.FailNowf(err.Error(), "invalid modelId for subscription test case %s", name)
	}

	return kvstoreSubscriptionTestCase{
		subscriptionTestCase: subscriptionTestCase{
			Name:    name,
			ModelId: modelId,
			Input:   input,
		},
		Assert: assert,
	}
}

type dbSubscriptionTestCase struct {
	subscriptionTestCase
	Assert DbSubscriptionAssertion
}

type DbSubscriptionAssertion func(t *testing.T, fetcher *DbSubscriptionFetcher)

type DbSubscriptionFetcher struct {
	t    *testing.T
	orm  *gorm.DB
	name string
}

func (f DbSubscriptionFetcher) ByPrimaryKey(key interface{}, model interface{}) {
	res := f.orm.First(model, key)
	assert.NoErrorf(f.t, res.Error, "unexpected error on fetching db subscription %s", f.name)
}

type kvstoreSubscriptionTestCase struct {
	subscriptionTestCase
	Assert KvStoreSubscriptionAssertion
}

type KvStoreSubscriptionAssertion func(t *testing.T, fetcher *KvstoreSubscriptionFetcher)

type KvstoreSubscriptionFetcher struct {
	t     *testing.T
	store kvstore.KvStore
	name  string
}

func (f KvstoreSubscriptionFetcher) Get(key interface{}, model interface{}) {
	ok, err := f.store.Get(context.Background(), key, model)

	assert.Truef(f.t, ok, "model for subscription %s and key %v should be available in the store", f.name, key)
	assert.NoErrorf(f.t, err, "unexpected error on fetching kvstore subscription %s", f.name)
}
