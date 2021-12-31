# Integration tests

In this section we will be looking at three types of integration tests provided by Gosoline: 'resty.Client', 'ApiServerTestCase', and 'StreamTestCase'. All three can be used together in the same test, and Gosoline's `suite` package is responsible for reading their configuration, starting and running an application, then running the actual tests cases. Below is their description:

### resty.Client integration tests

In the [Integration tests for your API](../getting_started/integration_tests.md) example, we saw the following test:

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

The `Test_ToEuro` method makes a GET call to an endpoint, in order to receive back the euro exchange value for 10 GBP. Lastly, it checks if the received value is 8.0.

In order to make this GET call `Test_ToEuro` does not need to concern itself about the IP on which the exchange application is running, nor its port. All `Test_ToEuro` needs to know is the URL path of that endpoint, as the _client_ object does the rest.

This _client_ object is provided by Gosoline, whenever at least one of a test suite's methods has the above signature.

[embedmd]:# (../../examples/getting_started/integration/api_test.go /func \(s \*ApiTestSuite\) SetupApiDefinitions/ /return definer.ApiDefiner\n}/)
```go
func (s *ApiTestSuite) SetupApiDefinitions() apiserver.Definer {
	return definer.ApiDefiner
}
```

Implementing _SetupApiDefinitions_ when configuring your test will inform Gosoline that an API is being tested, thus the _client_ object will contain meaningful data.

Forgeting to implement _SetupApiDefinitions_, and the trying to run an _resty.Client_ test case, will result in an error reminding you that `the suite has to implement the TestingSuiteApiDefinitionsAware interface`.

### ApiServer integration tests

The main advantage of the _client_ object is it allows you to control when requests are executed and gives you access to the endpoint, by providing the host & port. Therefore, if you, for example, want to time your requests, this is the way to go.  If you only want to make a standard API call and read the response, you can use Gosoline's _ApiServerTestCase_. These type of tests issue HTTP calls and compare their responses against a predefined value, but they need to follow a predefined structure. Below we can see how the above _resty client_ test would look like as an _ApiServerTestCase_:

[embedmd]:# (../../examples/getting_started/integration/api_test.go /func \(s \*ApiTestSuite\) Test_Euro/ /\n}/)
```go
func (s *ApiTestSuite) Test_Euro() *suite.ApiServerTestCase {
	return &suite.ApiServerTestCase{
		Method:             http.MethodGet,
		Url:                "/euro/10/GBP",
		Headers:            map[string]string{},
		ExpectedStatusCode: http.StatusOK,
		Assert: func(response *resty.Response) error {
			result, err := strconv.ParseFloat(string(response.Body()), 64)
			s.NoError(err)
			s.Equal(8.0, result)

			return nil
		},
	}
}
```

Notice the predefined structure of an _ApiServerTestCase_: it is an object that has fields for all the information needed in performing an HTTP call, the expected status code, and a method that tests the result for correctness.

In the example from `examples/getting_started/integration/api_test.go` we see two types of test cases (_resty.Client_ and _ApiServerTestCase_) in the same test suite. In fact, all three types of tests can be part of the same test suite.

### Stream integration tests

If you have a Gosoline application that takes its input from a stream, you need a test which can run your application locally, send data to a stream, and check your application's output for correctness. 

The stream-consumer application, found in more_details/stream-consumer, reads unsigned integers from the `consumerInput` input, increments them by one, and publishes them to the `publisher-outputEvent` output. Below is an extract from its integration test:

[embedmd]:# (../../examples/more_details/stream-consumer-test/stream_consumer_test.go /func \(s \*ConsumerTestSuite\) SetupSuite/ /\n}/)
```go
func (s *ConsumerTestSuite) SetupSuite() []suite.Option {
	return []suite.Option{
		suite.WithLogLevel("debug"),
		suite.WithConfigFile("../stream-consumer/config.dist.yml"),
		suite.WithModule("consumerModule", stream.NewConsumer("uintConsumer", consumer.NewConsumer())),
	}
}
```

It is making use of the same `config.dist.yml` file as `stream-consumer`, and it will be using the module created by _consumer.NewConsumer_. A _StreamTestCase_ is very similar to an _ApiServerTestCase_:

[embedmd]:# (../../examples/more_details/stream-consumer-test/stream_consumer_test.go /func \(s \*ConsumerTestSuite\) TestSuccess/ /\n}/)
```go
func (s *ConsumerTestSuite) TestSuccess() *suite.StreamTestCase {
	return &suite.StreamTestCase{
		Input: map[string][]suite.StreamTestCaseInput{
			"consumerInput": {
				{
					Attributes: nil,
					Body:       mdl.Uint(5),
				},
			},
		},
		Assert: func() error {
			var result int
			s.Env().StreamOutput("publisher-outputEvent").Unmarshal(0, &result)

			s.Equal(6, result)

			return nil
		},
	}
}
```

This _StreamTestCase_ is an object defining an input and an `Assert` function. Gosoline will run the module, use this `StreamTestCaseInput` as input for it, then run `Assert`. The `Assert` function reads the first element from a stream, _publisher-outputEvent_, then compares it with an expected result. Notice that the input has a key called `consumerInput`, because in this application's `config.dist.yml` file, we have configured an input named `consumerInput`.

## Shared environment

One of the options for a Gosoline _suite_ integration test is `suite.WithSharedEnvironment()`. When this option is off, each test case will run in its own environment. For example, the fixtures are being loaded for every test case, and any change to a database or stream will only last during that test case alone. When this option is enabled, the environment is created only once and used by all the test cases, and any change done by one test case will be available to the ones who follow.

## Auto detect components

Another option each _suite_ test offers is `WithoutAutoDetectedComponents`. This simply adds one extra options to the test, which tells it to skip one of the components configured in any potential `config.dist.yml`. The skipped component's name is given as a parameter to `WithoutAutoDetectedComponents`. Also note that while a component can be skipped by auto detect, it can still be added manually to the test via an option.
