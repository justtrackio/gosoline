## Give me well-formed messages

Publishing messages with our own structure can be useful when we are talking to services written in other languages or which are not using gosoline.
Maybe we are even writing to something like a kinesis stream which gets consumed by a firehose.
However, quite often we simply want to send a message from one service to another service and don't care too much about the final format as long as the other service gets back exactly what we sent it.

Let us first take a look how to use the `Producer` to publish a message:

[//]: # (01_producer_example/main.go,02_producer_compression_example/main.go)

```go
package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
)

func main() {
	app := application.Default()
	app.Add("producer-module", NewProducerModule)
	app.Run()
}

func NewProducerModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	modelId := mdl.ModelId{
		Name: "exampleEvent",
	}
	modelId.PadFromConfig(config)
	producer, err := stream.NewProducer(config, logger, "exampleProducer")

	if err != nil {
		return nil, fmt.Errorf("failed to create example producer: %w", err)
	}

	return producerModule{
		modelId:  modelId,
		producer: producer,
		logger:   logger,
	}, nil
}

type producerModule struct {
	modelId  mdl.ModelId
	producer stream.Producer
	logger   mon.Logger
}
```

[//]: # (01_producer_example/main.go)

The structure is quite similar to the one of the `Output` from the previous post.
The main difference is the additional `ModelId` we create and initialize from the config.
It will help us provide additional attributes on our message later so other services can determine what kind of message they are dealing with.

```go
func (m producerModule) Run(ctx context.Context) error {
	msg, err := stream.MarshalJsonMessage(map[string]interface{}{
		"greeting": "hello, world",
	}, mdlsub.CreateMessageAttributes(m.modelId, mdlsub.TypeCreate, 0))

	if err != nil {
		return err
	}

	return m.producer.WriteOne(ctx, msg)
}
```

We use `MarshalJsonMessage` to convert our message to JSON and `CreateMessageAttributes` to create attributes for the message.
This will convert our `ModelId` struct to the string `gosoline.stream-example.producer-example.exampleEvent`, telling a consumer of the message what kind of event it is dealing with.
We also get a `type` and a `version` attribute on the message which might help other services to handle messages of an old and a new format differently.
The complete message generated looks like this:

[//]: # (01_producer_example/message.json)

```json
{
  "attributes": {
    "encoding": "application/json",
    "modelId": "gosoline.stream-example.producer-example.exampleEvent",
    "type": "create",
    "version": 0
  },
  "body": "{\"greeting\":\"hello, world\"}"
}
```

Our JSON-encoded body was stored as a string in the `body` property.
While this may seem a little wasteful, it allows us to use different encodings in the body without tying ourselfs to JSON just because our container format is encoded as JSON. 

[//]: # (01_producer_example/config.dist.yml)

```yaml
env: test

app_project: gosoline
app_family: stream-example
app_name: producer-example

stream:
  output:
    exampleProducer:
      type: inMemory
```

If you look at the above configuration, you will notice (besides how the `app_project`, `app_family` and `app_name` fields got encoded in the `modelId` attribute) that we are still using the `stream.output` and not the `stream.producer` nesting.
The `Producer` is using an `Output`, so we need to tell it how to write messages somewhere.
We can however specify some additional settings for our producer using the `stream.producer.exampleProducer` nesting.
For example, we could tell Gosoline to compress our messages, to use a different encoding than JSON, a special output, or enable the producer daemon to batch messages together (given we accept that we might lose some messages if we can't write them out fast enough upon app exit).

## I want to laugh - a lot

So lets go ahead and produce a message containing around 1 million times the string `LOL`.
Such a message would be about 3 MB big without any compression.
Thus, we could not send it for example via SQS (if you don't put the message body in S3), leaving us dead in the water.
However, if we manage to compress that string, it shrinks to somewhere around 4 KB in size which we can transmit a lot easier.
So we start this time by enabling gzip compression for our producer:

[//]: # (02_producer_compression_example/config.dist.yml)

```yaml
env: test

app_project: gosoline
app_family: stream-example
app_name: producer-compression-example

stream:
  producer:
    exampleProducer:
      compression: application/gzip

  output:
    exampleProducer:
      type: inMemory
```

[//]: # (02_producer_compression_example/main.go)

We start by generating a long string to test with.
As we said, we want to generate 1 million times `LOL`, so the following code should take care of that by generating 2<sup>20</sup> times `LOL`:

```go
func oneMillionLOLs() string {
	lol := "LOL"

	for i := 0; i < 20; i++ {
		lol = lol + lol
	}

	return lol
}
```

We can now construct our message.
To observe the compression actually taking place we can take the logger I already sneaked in for the last version of the app:

```go
func (m producerModule) Run(ctx context.Context) error {
	greeting := oneMillionLOLs()
	msg, err := stream.MarshalJsonMessage(map[string]interface{}{
		"greeting": greeting,
	}, mdlsub.CreateMessageAttributes(m.modelId, mdlsub.TypeCreate, 0))

	if err != nil {
		return err
	}

	m.logger.WithContext(ctx).Info("publishing a message with more than %d characters", len(greeting))

	defer func() {
		output := stream.ProvideInMemoryOutput("exampleProducer")
		msg, ok := output.Get(0)

		if ok {
			m.logger.WithContext(ctx).Info("published message with encoded body length of %d characters, attributes %v", len(msg.Body), msg.Attributes)
		}
	}()

	return m.producer.WriteOne(ctx, msg)
}
```

It should now print two messages like these:

    publishing a message with more than 3145728 characters
    published message with encoded body length of 4304 characters, attributes map[compression:application/gzip encoding:application/json]

You can see that we now have two attributes attached to the message.
Those are not the attributes we already saw before (those are actually part of the encoded body), but the attributes which get send to a service like SNS or SQS.
Gosoline then uses those when receiving the message to decompress it before it can access the attributes useful for message routing (like the `modelId` attribute).

## I don't want to parse messages by hand

Publishing messages helps us a lot more when we can consume them at some other part again.
Gosoline can abstract most of the process of consuming for us.
That includes fetching messages from time to time, parsing them into some structure for us, and acknowledging a message once we processed it successfully.
Gosoline provides a special application type for consumers, so we can use `application.RunConsumer` with a `stream.ConsumerCallback`:

[//]: # (03_consumer_example/main.go)

```go
package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
	"github.com/applike/gosoline/pkg/stream"
	"time"
)

func main() {
	application.RunConsumer(NewConsumerCallback)
}

func NewConsumerCallback(_ context.Context, _ cfg.Config, logger mon.Logger) (stream.ConsumerCallback, error) {
	go provideFakeData()

	return consumerCallback{
		logger: logger,
	}, nil
}

type consumerCallback struct {
	logger mon.Logger
}
```

As you can see our consume is not using a lot of dependencies (only a single logger) for this example.
We will be implementing a small application which can either print something or wait for some time.
Therefore `provideFakeData` will write some messages to our input and basically play the role of some external service publishing messages.
Each message contains a body which will be parsed into some go struct and a modelId attribute we can use in `GetModel` to return the struct we want to parse our body into:

```go
func provideFakeData() {
	input := stream.ProvideInMemoryInput("exampleInput", &stream.InMemorySettings{
		Size: 1,
	})

	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.printCommand",
		},
		Body: `{"message":"hello, world"}`,
	})
	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.waitCommand",
		},
		Body: `{"time":1}`,
	})
	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.printCommand",
		},
		Body: `{"message":"processing..."}`,
	})
	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.waitCommand",
		},
		Body: `{"time":3}`,
	})
	input.Publish(&stream.Message{
		Attributes: map[string]interface{}{
			"modelId": "gosoline.stream-example.example.printCommand",
		},
		Body: `{"message":"bye"}`,
	})

	input.Stop()
}
```

When we get a new message, Gosoline calls `GetModel` with the attributes of that message.
We can use it to return the correct struct for each message.
Most consumers will just consume one type of message and therefore don't need to inspect the attributes but as we see this does not have to be the case.

```go
type PrintCommand struct {
	Message string `json:"message"`
}

type WaitCommand struct {
	Time int `json:"time"`
}

func (c consumerCallback) GetModel(attributes map[string]interface{}) interface{} {
	switch attributes["modelId"] {
	case "gosoline.stream-example.example.printCommand":
		return &PrintCommand{}
	case "gosoline.stream-example.example.waitCommand":
		return &WaitCommand{}
	default:
		return nil
	}
}
```

We can finally implement our callback to handle the record we are currently consuming.
Depending on the type of command we received from Gosoline in this callback we either log the message or sleep for some time.
Let us take a moment to look at the other unused arguments for this callback:
We receive our attributes a second time. So we could use the model id from the attributes instead of the type of the model itself to determine what to do with the message.
We also receive a context which we should use to perform any IO like publishing new messages, reading or writing to a database, etc.
Our callback then returns two values.
A boolean indicating whether we want to acknowledge (delete from the queue) the message.
If you are consuming a queue which can not acknowledge messages, the value you return there is ignored.
An error telling gosoline whether you were successful when processing the message.
Note that you can still acknowledge messages even in the case of failure (if you know you can never handle the message), and you don't have to acknowledge messages if you were successful (you just want to process the message again after e.g. the visibility timeout of your SQS queue elapsed).

```go
func (c consumerCallback) Consume(ctx context.Context, model interface{}, attributes map[string]interface{}) (bool, error) {
	switch cmd := model.(type) {
	case *PrintCommand:
		c.logger.WithContext(ctx).Info("printing message: %s", cmd.Message)

		return true, nil
	case *WaitCommand:
		time.Sleep(time.Duration(cmd.Time) * time.Second)

		return true, nil
	default:
		return true, fmt.Errorf("unknown model: %s with type %T", attributes["modelId"], model)
	}
}
```

Let us finally declare our config for our consumer:

[//]: # (03_consumer_example/config.dist.yml)

```yaml
env: test

app_project: gosoline
app_family: stream-example
app_name: consumer-example

stream:
  consumer:
    default:
      input: exampleInput
      runner_count: 1
  input:
    exampleInput:
      type: inMemory
      size: 1
```

Not much interesting stuff is happening here.
We define a consumer called `default` (the name given to it by `application.RunConsumer`).
We name the input of the consumer and give it a definition.
We also define the `runner_count` for the consumer.
Each runner is a go routine which is calling `GetModel` and `Consume`, so if you declare a `runner_cout` of 5, up to 5 runners can call `Consume` at the same time (so it better be thread safe).
Having only a single runner therefore is the simplest configuration, but it might make it easier to decide how many instances of your service you need to run.

This concludes the overview of producers and consumers.
[Next](../03_stream_message_attributes_subscribers_and_the_producer_daemon/index.md) time we will look at another abstraction on top of producers as well as a specialized kind of consumer - the subscriber. 
