# A tour through gosoline - publishing and receiving messages

Gosoline is an opinionated framework aimed at building small microservices which communicate via HTTP calls or publish messages to each other.
We will therefore start our tour through the different parts of gosoline by looking at how a service can publish messages for other services and consume them again.
We will start with `Input`s and `Output`s as primitives to send messages, then work our way up to `Producer`s and `Consumer`s as a way to abstract away the message format
and finally look at `Subscriber`s and `Publisher`s for some quite high-level abstractions provided by gosoline.

## Writing to an output

[//]: # (01_output_example/main.go)

We first start by publishing a message to an `Output`.
An `Output` allows us to publish anything to it which can encode itself to some utf-8 string.
You would normally not use an `Output` directly, but it is often useful to know what drives your code under the hood. 
Gosoline supports quite some output types like writing to a file, collecting messages in memory (useful for unit tests),
writing to a [Kinesis](https://docs.aws.amazon.com/streams/latest/dev/introduction.html) stream,
writing to an [SQS](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/welcome.html) queue
or an [SNS](https://docs.aws.amazon.com/sns/latest/dg/welcome.html) topic,
or writing to a [Redis](https://redis.io/commands/rpush) list.
We will make use of the `inMemory` `Output` as it requires no additional setup or external dependencies.

We start by creating a new default application, add a new module and run the app: 

```go
package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

func main() {
	app := application.Default()
	app.Add("output-module", NewOutputModule)
	app.Run()
}
```

Next, we define the kernel module containing our logic.
We then create a new configurable output with the name `exampleOutput`.
Using a configurable output instead of hard coding the nature of our output makes it easier to switch the implementation later on, for example in an integration test.
Finally, we package everything up in a module the kernel can run: 

```go
func NewOutputModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	output, err := stream.NewConfigurableOutput(config, logger, "exampleOutput")

	if err != nil {
		return nil, fmt.Errorf("failed to create example output: %w", err)
	}

	return outputModule{
		output: output,
	}, nil
}

type outputModule struct {
	output stream.Output
}
```

Eventually we now have all the pieces in place we need to send a single message.
Our message will just have the body `{"greeting":"hello, world"}` for now.
After creating the message, we pass it to `WriteOne` on our `Output`.
If everything goes well we will get a `nil` error back and can return that to our caller to signal everything went well.
If you ever have more than one message to publish, there is also a `Write` function on the `Output` which accepts more than one message at once.
It is more efficient on some output types (like SQS) as it can send multiple messages per API call.
Other outputs such as in-memory (or, maybe more realistically, SNS) outputs don't benefit from the use of batch writes that much, but it will not hurt them either.

```go
func (m outputModule) Run(ctx context.Context) error {
	msg := stream.NewRawJsonMessage(map[string]interface{}{
		"greeting": "hello, world",
	})

	return m.output.WriteOne(ctx, msg)
}
```

As we defined our output as a configurable output we have to define its type and any needed additional settings in the `config.dist.yml` file.
We also have to define the project, family and name of our app as well as the environment we are running in:

[//]: # (01_output_example/config.dist.yml)

```yml
env: test

app_project: gosoline
app_family: stream-example
app_name: output-example

stream:
  output:
    exampleOutput:
      type: inMemory
```

Now we can use `go run` to test our application.
It will do not much we can see yet, but we will change that in the next section.

## Reading from an input

[//]: # (02_input_example/main.go)

Now that we can publish messages it is time to consume some messages as well.
We will make use of the `inMemory` `Input` as it allows us to fake some messages without depending on any external services.
We start again by creating an application with a single module which will consume the messages we get from the `Input`:

```go
package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/coffin"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"time"
)

func main() {
	app := application.Default()
	app.Add("input-module", NewInputModule)
	app.Run()
}
```

Next we initialize the module and create out `Input`.
Similar to the `Output` we use a configurable `Input` to allow us to configure the input in the `config.dist.yml`.
Indeed, for every `Output` type there is a corresponding `Input` type (an SNS input reads from SQS but expects a different format than an SQS input).
After creating the input we also start a go routine to provide some fake data for us to work with.
It looks up the input by name and publishes 10 `Message`s to it, each having a slightly longer body of `a`s.
The `stream.Message` struct is our main container format and almost all messages you will be working with will be embedded in that container.
It allows us to add some attributes to a message which in turn enable gosoline to provide features like message compression or batching multiple messages together.
You could also use it to receive messages from many sources and distinguish them while reading from a single input.
 
```go
func NewInputModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	input, err := stream.NewConfigurableInput(config, logger, "exampleInput")

	if err != nil {
		return nil, fmt.Errorf("failed to create example input: %w", err)
	}

	go provideFakeData()

	return inputModule{
		input:  input,
		logger: logger,
	}, nil
}

type inputModule struct {
	input  stream.Input
	logger mon.Logger
}

func provideFakeData() {
	input := stream.ProvideInMemoryInput("exampleInput", &stream.InMemorySettings{
		Size: 1,
	})

	for msg := "a"; len(msg) <= 10; msg += "a" {
		input.Publish(&stream.Message{
			Body: msg,
		})
	}
}
```

Finally, we have everything in place to start reading from our `Input`.
We just need to call `Run` on it and we will have each message available on the channel obtained from `Data`.
Just one little problem: `Run` does not return until we call `Stop` on the input (or the context is canceled).
Luckily for us we can easily fork another go routine to handle driving the input while we consume the messages in peace.
If we however want to track any errors thrown by `Run`, we need to capture them somehow.
Gosoline provides some easy to use abstraction for that allows us to track multiple go routines and collect the errors they return called a `Coffin`.
Thus, we use `coffin.New()` to create one, fork the `Run` method in it, fork our consumer in it, and wait for both routines to return.

Finally, we can get to business and consume some messages.
We iterate over all messages provided by `Data()`, perform some "work" for each message and if we consumed 10 messages, stop the input.
Stopping the input will cause no new messages to be published to the channel (but there still might be some messages left over afterwards), so after calling `Stop()` we will eventually exit our loop and return.

```go
func (m inputModule) Run(ctx context.Context) error {
	logger := m.logger.WithContext(ctx)
	cfn := coffin.New()

	cfn.GoWithContext(ctx, m.input.Run)
	cfn.Go(func() error {
		consumed := 0

		for item := range m.input.Data() {
			logger.Info("received new message, processing it: %s", item.Body)

			// fake some work...
			time.Sleep(time.Millisecond * 100 * time.Duration(len(item.Body)))

			consumed++

			if consumed == 10 {
				m.input.Stop()
			}
		}

		return nil
	})

	return cfn.Wait()
}
```

Similar to the `Output` we have to define our `Input` in the `config.dist.yml`.
An `inMemory` input also accepts an additional config parameter `size` for the buffer of the channel used internally.
It defaults to `1`, but we specify it for completeness:

[//]: # (02_input_example/config.dist.yml)

```yml
env: test

app_project: gosoline
app_family: stream-example
app_name: input-example

stream:
  input:
    exampleInput:
      type: inMemory
      size: 1
```

When we now `go run` the application, we see how we consume messages at an increasingly slower rate (as messages grow larger) until after the 10th message our application shuts down again.
This concludes the first overview of inputs and outputs.
[Next](../02_stream_well_formed_messages_and_the_consumer_framework/index.md) time we will take a look on the first abstractions around inputs and outputs and how we can make use of it to build distributed applications.
