# Integration tests for your API

The _money-exchange_ application exposes two endpoints, `euro` and `euro-at-date`, which can convert an amount from a given currency to its equivalent in euros, or to its equivalent in euros using the exchange rate at a given date, respectively.

To test this application, one needs to be able to issue calls to both endpoints, and check their results for correctness. Luckily Gosoline offers plenty of help with this.

### Package _suite.Suite_

This package acts as a wrapper over Golang's _testing_, and each _Suite_ object implements the following interface:

[embedmd]:# (../../pkg/test/suite/suite.go /type TestingSuite interface/ /\n}/)
```go
type TestingSuite interface {
	Env() *env.Environment
	SetEnv(environment *env.Environment)
	SetT(t *testing.T)
	T() *testing.T
	SetupSuite() []Option
}
```

The starting point for a Gosoline integration test is the `suite.Run` function:

[embedmd]:# (../../pkg/test/suite/run.go /func Run/ /\)/)
```go
func Run(t *testing.T, suite TestingSuite, extraOptions ...Option)
```

This method will use reflection to find all test cases declared by a given _TestingSuite_ object, apply each of the _extraOptions_, create a kernel with whatever modules or APIs were declared in the `SetupSuite` and `SetupApiDefinitions`, and lastly run the tests.

Each Gosoline integration test follows the same format:

- Creates an object which implements _TestingSuite_
- Implements the `SetupSuite` method for that object
- Has at least one `Test...` method
- It calls `suite.Run`

### Practical example

Inside `examples/getting_started/integration` we have an integration test for _money-exchange_. It is comprised out of two files, `api_test.go` and `fixtures.go`.

`fixtures.go` contains initial values for the `currency` key value store. Its most important part is:

[embedmd]:# (../../examples/getting_started/integration/fixtures.go /var fixtureSets/ /\n}/)
```go
var fixtureSets = []*gosoFixtures.FixtureSet{
	{
		Enabled: true,
		Writer:  gosoFixtures.ConfigurableKvStoreFixtureWriterFactory("currency"),
		Fixtures: []interface{}{
			&gosoFixtures.KvStoreFixture{
				Key:   "GBP",
				Value: 1.25,
			},
			&gosoFixtures.KvStoreFixture{
				Key:   "2021-01-03-GBP",
				Value: 0.8,
			},
		},
	},
}
```

This _FixtureSet_ holds exchange rate data for GBP, which will be used by the test. The purpose of fixtures is to predefine test data, because we do not want to download actual exchange rates just for one test. 

`api_test.go` declares _ApiTestSuite_, which will implement _TestingSuite_ :

[embedmd]:# (../../examples/getting_started/integration/api_test.go /type ApiTestSuite/ /\n}/)
```go
type ApiTestSuite struct {
	suite.Suite

	clock clock.Clock
}
```

It also declares a single, normal, unit test. This unit test makes use of the above struct and calls _suite.Run_:

[embedmd]:# (../../examples/getting_started/integration/api_test.go /func TestApiTestSuite/ /\n}/)
```go
func TestApiTestSuite(t *testing.T) {
	suite.Run(t, &ApiTestSuite{
		clock: clock.NewFakeClockAt(time.Now().UTC()),
	})
}
```

Notice the use of `clock.NewFakeClockAt`. This is an elegant solution to the following problem: when testing the same code multiple times, we want the test results to be identical. For code that makes calls to `time.Now()` this is by default not true. Using a fake clock, which always returns a predefined time, solves this problem. 

Another important thing about _ApiTestSuite_ is how it configures the tests, by implementing `SetupSuite`:

[embedmd]:# (../../examples/getting_started/integration/api_test.go /func \(s \*ApiTestSuite\) SetupSuite/ /\n}/)
```go
func (s *ApiTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("info"),
		suite.WithConfigFile("../api/config.dist.yml"),
		suite.WithFixtures(fixtureSets),
		suite.WithClockProvider(s.clock),
	}
}
```

In `SetupSuite` it makes use of the same `config.dist.yml` file that _money-exchange_ uses, and also of the fixture data declared in `fixtures.go`.

The api server itself is configured by `SetupApiDefinitions`:

[embedmd]:# (../../examples/getting_started/integration/api_test.go /func \(s \*ApiTestSuite\) SetupApiDefinitions/ /\n}/)
```go
func (s *ApiTestSuite) SetupApiDefinitions() apiserver.Definer {
	return definer.ApiDefiner
}
```

Lastly, for each method of _ApiTestSuite_ starting with `Test...`, Gosoline will run a separate test case. In this example, we have two such methods:

[embedmd]:# (../../examples/getting_started/integration/api_test.go /func \(s \*ApiTestSuite\) Test_ToEuro/ /\n}/)
```go
func (s *ApiTestSuite) Test_ToEuro(_ suite.AppUnderTest, client *resty.Client) error {
	var result float64

	response, err := client.R().
		SetResult(&result).
		Execute(http.MethodGet, "/euro/10/GBP")

	s.NoError(err)
	s.Equal(http.StatusOK, response.StatusCode())
	s.Equal(8.0, result)

	return nil
}
```

and

[embedmd]:# (../../examples/getting_started/integration/api_test.go /func \(s \*ApiTestSuite\) Test_ToEuroAtDate/ /\n}/)
```go
func (s *ApiTestSuite) Test_ToEuroAtDate(_ suite.AppUnderTest, client *resty.Client) error {
	var result float64

	response, err := client.R().
		SetResult(&result).
		Execute(http.MethodGet, "/euro-at-date/10/GBP/2021-01-03T00:00:00Z")

	s.NoError(err)
	s.Equal(http.StatusOK, response.StatusCode())
	s.Equal(12.5, result)

	return nil
}
```

Both `Test_ToEuro` and `Test_ToEuroAtDate` will issue a REST call to the API server, and check its response for correctness. 

`Test_ToEuro` uses `client` to make an http GET call. Then it uses `Equal` and `NoError` to test its correctness. _ApiTestSuite_ offers `Equal` and `NoError` becuase it embeds `suite.Suite`, and `suite.Suite` embeds `*assert.Assertions`.

In a similar manner, `Test_ToEuroAtDate` calls the `euro-at-date` endpoint and checks the result for correctness.

To run the test, navigate to the file with a terminal, and type `go test . --tags integration,fixtures -v`

The tags are needed because the test file starts with:

```go
//go:build integration && fixtures
// +build integration,fixtures
```

 ### Wrapping it up

Gosoline's suite package is meant to make writing integration tests easier and faster. For a web application composed out of many microservices, aim to have at least one integration test for each microservice, ideally one test for every use case.
