package suite

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/justtrackio/gosoline/pkg/application"
	"github.com/justtrackio/gosoline/pkg/db-repo"
	"github.com/justtrackio/gosoline/pkg/ddb"
	"github.com/justtrackio/gosoline/pkg/kvstore"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/mdlsub"
	"github.com/justtrackio/gosoline/pkg/test/env"
	"github.com/stretchr/testify/assert"
)

func init() {
	registerTestCaseDefinition("subscriber", isTestCaseSubscriber, buildTestCaseSubscriber)
}

type SubscriberTestCase interface {
	GetName() string
	GetModelId() mdl.ModelId
	GetInput() any
	GetVersion() int
}

type subscriberTestCase struct {
	Name    string
	ModelId mdl.ModelId
	Input   any
	Output  mdlsub.Model
	Version int
}

func (s subscriberTestCase) GetName() string {
	return s.Name
}

func (s subscriberTestCase) GetModelId() mdl.ModelId {
	return s.ModelId
}

func (s subscriberTestCase) GetInput() any {
	return s.Input
}

func (s subscriberTestCase) GetVersion() int {
	return s.Version
}

const expectedTestCaseSubscriberSignature = "func (s TestingSuite) TestFunc() (SubscriberTestCase, error)"

func isTestCaseSubscriber(method reflect.Method) error {
	if method.Func.Type().NumIn() != 1 {
		return fmt.Errorf("expected %q, but function has %d arguments", expectedTestCaseSubscriberSignature, method.Func.Type().NumIn())
	}

	if method.Func.Type().NumOut() != 2 {
		return fmt.Errorf("expected %q, but function has %d return values", expectedTestCaseSubscriberSignature, method.Func.Type().NumOut())
	}

	actualType0 := method.Func.Type().In(0)
	expectedType0 := reflect.TypeOf((*TestingSuite)(nil)).Elem()

	if !actualType0.Implements(expectedType0) {
		return fmt.Errorf("expected %q, but first argument type/receiver type is %s", expectedTestCaseSubscriberSignature, actualType0.String())
	}

	actualTypeResult0 := method.Func.Type().Out(0)
	expectedTypeResult0 := reflect.TypeOf((*SubscriberTestCase)(nil)).Elem()

	if actualTypeResult0 != expectedTypeResult0 && !actualTypeResult0.Implements(expectedTypeResult0) {
		return fmt.Errorf("expected %q, but first return type is %s", expectedTestCaseSubscriberSignature, actualTypeResult0.String())
	}

	actualTypeResult1 := method.Func.Type().Out(1)
	expectedTypeResult1 := reflect.TypeOf((*error)(nil)).Elem()

	if actualTypeResult1 != expectedTypeResult1 {
		return fmt.Errorf("expected %q, but second return type is %s", expectedTestCaseSubscriberSignature, actualTypeResult1.String())
	}

	return nil
}

func buildTestCaseSubscriber(_ TestingSuite, method reflect.Method) (testCaseRunner, error) {
	return func(t *testing.T, suite TestingSuite, suiteOptions *suiteOptions, environment *env.Environment) {
		suite.SetT(t)

		suiteOptions.addAppOption(application.WithConfigMap(map[string]any{
			"httpserver": map[string]any{
				"default": map[string]any{
					"port": 0,
				},
			},
		}))

		runTestCaseApplication(t, suite, suiteOptions, environment, func(app *appUnderTest) {
			ret := method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
			err := ret[1].Interface()
			if err != nil {
				assert.FailNow(t, err.(error).Error())
			}

			tc := ret[0].Interface().(SubscriberTestCase)

			attrs := mdlsub.CreateMessageAttributes(tc.GetModelId(), "create", tc.GetVersion())

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

				orm, err := db_repo.NewOrm(suite.Env().Context(), config, logger, "default")
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
					repo: func(model any) (ddb.Repository, error) {
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

				store, err := kvstore.NewConfigurableKvStore[mdlsub.Model](ctx, config, logger, tc.GetName())
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
			Version: testCase.Version,
		},
		Assert: testCase.Assert,
	}, nil
}

type DbSubscriberTestCase struct {
	Name    string
	ModelId string
	Version int
	Input   any
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

func (f DbSubscriberFetcher) ByPrimaryKey(key any, model any, options ...SubscriptionFetcherOption) {
	opts := runSubscriptionFetcherOptions(options)

	res := f.orm.First(model, key)

	if gorm.IsRecordNotFoundError(res.Error) {
		assert.Falsef(f.t, opts.expectFound, "unexpected missing item with key %v when fetching db subscription %s", key, f.name)

		return
	}

	assert.NoErrorf(f.t, res.Error, "unexpected error on fetching db subscription %s", f.name)

	if res.Error != nil {
		return
	}

	assert.Truef(f.t, opts.expectFound, "unexpected found item with key %v when fetching db subscription %s", key, f.name)
}

type SubscriptionFetcherOption func(*subscriberFetcherOptions)

type subscriberFetcherOptions struct {
	expectFound bool
}

func WithExpectFound(o *subscriberFetcherOptions) {
	o.expectFound = true
}

func WithExpectNotFound(o *subscriberFetcherOptions) {
	o.expectFound = false
}

func runSubscriptionFetcherOptions(options []SubscriptionFetcherOption) subscriberFetcherOptions {
	opts := subscriberFetcherOptions{
		expectFound: true,
	}

	for _, option := range options {
		option(&opts)
	}

	return opts
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
			Version: testCase.Version,
		},
		ModelIdTarget: modelIdTarget,
		Assert:        testCase.Assert,
	}, nil
}

type DdbSubscriberTestCase struct {
	Name          string
	SourceModelId string
	TargetModelId string
	Version       int
	Input         any
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
	repo func(model any) (ddb.Repository, error)
	name string
}

func (f DdbSubscriberFetcher) ByHash(hash any, model any, options ...SubscriptionFetcherOption) {
	opts := runSubscriptionFetcherOptions(options)

	repo, err := f.repo(model)
	assert.NoErrorf(f.t, err, "unexpected error on fetching ddb subscription %s", f.name)

	qb := repo.GetItemBuilder().WithHash(hash)
	res, err := repo.GetItem(context.Background(), qb, model)

	assert.NoErrorf(f.t, err, "unexpected error on fetching ddb subscription %s", f.name)

	if err != nil {
		return
	}

	if opts.expectFound {
		assert.Truef(f.t, res.IsFound, "unexpected missing item with hash %v when fetching ddb subscription %s", hash, f.name)
	} else {
		assert.Falsef(f.t, res.IsFound, "unexpected found item with hash %v when fetching ddb subscription %s", hash, f.name)
	}
}

func (f DdbSubscriberFetcher) ByHashAndRange(hash any, rangeValue any, model any, options ...SubscriptionFetcherOption) {
	opts := runSubscriptionFetcherOptions(options)

	repo, err := f.repo(model)
	assert.NoErrorf(f.t, err, "unexpected error on fetching ddb subscription %s", f.name)

	qb := repo.GetItemBuilder().WithHash(hash).WithRange(rangeValue)
	res, err := repo.GetItem(context.Background(), qb, model)

	assert.NoErrorf(f.t, err, "unexpected error on fetching ddb subscription %s", f.name)

	if err != nil {
		return
	}

	if opts.expectFound {
		assert.Truef(f.t, res.IsFound, "unexpected missing item with hash %v and range %v when fetching ddb subscription %s", hash, rangeValue, f.name)
	} else {
		assert.Falsef(f.t, res.IsFound, "unexpected found item with hash %v and range %v when fetching ddb subscription %s", hash, rangeValue, f.name)
	}
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
			Version: testCase.Version,
		},
		Assert: testCase.Assert,
	}, nil
}

type KvstoreSubscriberTestCase struct {
	Name    string
	ModelId string
	Input   any
	Version int
	Assert  KvStoreSubscriberAssertion
}

type kvstoreSubscriberTestCase struct {
	subscriberTestCase
	Assert KvStoreSubscriberAssertion
}

type KvStoreSubscriberAssertion func(t *testing.T, fetcher *KvstoreSubscriberFetcher)

type KvstoreSubscriberFetcher struct {
	t     *testing.T
	store kvstore.KvStore[mdlsub.Model]
	name  string
}

func (f KvstoreSubscriberFetcher) Get(key any, model mdlsub.Model, options ...SubscriptionFetcherOption) {
	opts := runSubscriptionFetcherOptions(options)

	ok, err := f.store.Get(context.Background(), key, &model)

	assert.NoErrorf(f.t, err, "unexpected error on fetching kvstore subscription %s", f.name)
	if err != nil {
		return
	}

	if opts.expectFound {
		assert.Truef(f.t, ok, "unexpected missing item with key %v when fetching kvstore subscription %s", key, f.name)
	} else {
		assert.Falsef(f.t, ok, "unexpected found item with key %v when fetching kvstore subscription %s", key, f.name)
	}
}
