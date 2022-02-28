package suite

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
)

func init() {
	testCaseDefinitions["subscriber"] = testCaseDefinition{
		matcher: isTestCaseSubscriber,
		builder: buildTestCaseSubscriber,
	}
}

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

func isTestCaseSubscriber(method reflect.Method) bool {
	if method.Func.Type().NumIn() != 1 {
		return false
	}

	if method.Func.Type().NumOut() != 2 {
		return false
	}

	var v SubscriberTestCase
	subscriberTestCaseType := reflect.ValueOf(&v).Elem().Type()
	arg0 := method.Func.Type().Out(0)

	if !arg0.Implements(subscriberTestCaseType) {
		return false
	}

	var err error
	errType := reflect.ValueOf(&err).Elem().Type()
	arg1 := method.Func.Type().Out(1)

	return arg1.Implements(errType)
}

func buildTestCaseSubscriber(_ TestingSuite, method reflect.Method) (testCaseRunner, error) {
	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		suite.SetT(t)

		ret := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
		tc := ret[0].Interface().(SubscriberTestCase)
		err := ret[1].Interface()
		if err != nil {
			assert.FailNow(t, err.(error).Error())
		}

		runTestCaseApplication(t, suite, suiteOptions, environment, func(app *appUnderTest) {
			attrs := mdlsub.CreateMessageAttributes(tc.GetModelId(), "create", 0)

			sourceModel := mdlsub.UnmarshalSubscriberSourceModel(suite.Env().Config(), tc.GetName())
			inputName := mdlsub.GetSubscriberFQN(tc.GetName(), sourceModel)
			suite.Env().StreamInput(inputName).PublishAndStop(tc.GetInput(), attrs)

			app.Stop()
			app.WaitDone()

			config := suite.Env().Config()
			logger := suite.Env().Logger()

			configKey := mdlsub.GetSubscriberOutputConfigKey(tc.GetName())
			outputType := config.GetString(configKey)

			switch outputType {
			case mdlsub.OutputTypeDb:
				dbSub, ok := tc.(dbSubscriberTestCase)

				if !ok {
					assert.FailNow(t, "invalid subscription test case", "the test case for the subscription of %s has to be of the db type", tc.GetName())
					return
				}

				orm, err := db_repo.NewOrm(config, logger)
				if err != nil {
					assert.FailNow(t, "can't initialize orm", "the test case for the subscription of %s can't be initialized", tc.GetName())
				}

				fetcher := &DbSubscriberFetcher{
					t:    t,
					orm:  orm,
					name: tc.GetName(),
				}

				dbSub.Assert(t, fetcher)

			case mdlsub.OutputTypeDdb:
				ctx := environment.Context()
				ddbSub, ok := tc.(ddbSubscriberTestCase)

				if !ok {
					assert.FailNow(t, "invalid subscription test case", "the test case for the subscription of %s has to be of the ddb type", tc.GetName())
					return
				}

				fetcher := &DdbSubscriberFetcher{
					t: t,
					repo: func(model interface{}) (ddb.Repository, error) {
						return ddb.NewRepository(ctx, config, logger, &ddb.Settings{
							ModelId: ddbSub.ModelIdTarget,
							Main: ddb.MainSettings{
								Model:              model,
								ReadCapacityUnits:  5,
								WriteCapacityUnits: 5,
							},
						})
					},
					name: tc.GetName(),
				}

				ddbSub.Assert(t, fetcher)

			case mdlsub.OutputTypeKvstore:
				ctx := environment.Context()
				dbSub, ok := tc.(kvstoreSubscriberTestCase)

				if !ok {
					assert.FailNow(t, "invalid subscription test case", "the test case for the subscription of %s has to be of the kvstore type", tc.GetName())
					return
				}

				store, err := kvstore.NewConfigurableKvStore(ctx, config, logger, tc.GetName())
				if err != nil {
					assert.FailNow(t, err.Error(), "the test case for the subscription of %s can't be initialized", tc.GetName())
				}

				fetcher := &KvstoreSubscriberFetcher{
					t:     t,
					store: store,
					name:  tc.GetName(),
				}

				dbSub.Assert(t, fetcher)
			}
		})
	}, nil
}

func DbTestCase(testCase DbSubscriberTestCase) (SubscriberTestCase, error) {
	modelId, err := mdl.ModelIdFromString(testCase.ModelId)
	if err != nil {
		return nil, fmt.Errorf("invalid modelId for subscription test case %s: %w", testCase.Name, err)
	}

	return dbSubscriberTestCase{
		subscriberTestCase: subscriberTestCase{
			Name:    testCase.Name,
			ModelId: modelId,
			Input:   testCase.Input,
		},
		Assert: testCase.Assert,
	}, nil
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

func DdbTestCase(testCase DdbSubscriberTestCase) (SubscriberTestCase, error) {
	var err error
	var modelIdSource, modelIdTarget mdl.ModelId

	if modelIdSource, err = mdl.ModelIdFromString(testCase.SourceModelId); err != nil {
		return nil, fmt.Errorf("invalid source modelId for subscription test case %s: %w", testCase.Name, err)
	}

	if modelIdTarget, err = mdl.ModelIdFromString(testCase.TargetModelId); err != nil {
		return nil, fmt.Errorf("invalid target modelId for subscription test case %s: %w", testCase.Name, err)
	}

	return ddbSubscriberTestCase{
		subscriberTestCase: subscriberTestCase{
			Name:    testCase.Name,
			ModelId: modelIdSource,
			Input:   testCase.Input,
		},
		ModelIdTarget: modelIdTarget,
		Assert:        testCase.Assert,
	}, nil
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
	repo func(model interface{}) (ddb.Repository, error)
	name string
}

func (f DdbSubscriberFetcher) ByHash(hash interface{}, model interface{}) {
	repo, err := f.repo(model)
	assert.NoErrorf(f.t, err, "unexpected error on fetching ddb subscription %s", f.name)

	qb := repo.GetItemBuilder().WithHash(hash)
	res, err := repo.GetItem(context.Background(), qb, model)

	assert.NoErrorf(f.t, err, "unexpected error on fetching ddb subscription %s", f.name)

	if err != nil {
		return
	}

	assert.True(f.t, res.IsFound)
}

func (f DdbSubscriberFetcher) ByHashAndRange(hash interface{}, rangeValue interface{}, model interface{}) {
	repo, err := f.repo(model)
	assert.NoErrorf(f.t, err, "unexpected error on fetching ddb subscription %s", f.name)

	qb := repo.GetItemBuilder().WithHash(hash).WithRange(rangeValue)
	res, err := repo.GetItem(context.Background(), qb, model)

	assert.NoErrorf(f.t, err, "unexpected error on fetching ddb subscription %s", f.name)

	if err != nil {
		return
	}

	assert.True(f.t, res.IsFound)
}

func KvstoreTestCase(testCase KvstoreSubscriberTestCase) (SubscriberTestCase, error) {
	modelId, err := mdl.ModelIdFromString(testCase.ModelId)
	if err != nil {
		return nil, fmt.Errorf("invalid modelId for subscription test case %s: %w", testCase.Name, err)
	}

	return kvstoreSubscriberTestCase{
		subscriberTestCase: subscriberTestCase{
			Name:    testCase.Name,
			ModelId: modelId,
			Input:   testCase.Input,
		},
		Assert: testCase.Assert,
	}, nil
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
