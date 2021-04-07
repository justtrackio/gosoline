## Give me proper attributes

As we have seen in the example for the producer, it can be quite tedious to add all the correct attributes to a message.
Luckily for us, gosoline provides another abstraction, called a `Publisher`.
A `Publisher` is quite similar to a `Producer`, but it always provides the `modelId`, `type`, and `version` attributes.

Let us start with a small program producing our greeting again.
First some boilerplate code:

[//]: # (01_publisher_example/main.go,02_producer_daemon_example/main.go)

```go
package main

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/mon"
)

func main() {
	app := application.Default()
	app.Add("publisher-module", NewProducerModule)
	app.Run()
}

func NewProducerModule(_ context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
	publisher, err := mdlsub.NewPublisher(config, logger, "examplePublisher")

	if err != nil {
		return nil, fmt.Errorf("failed to create example publisher: %w", err)
	}

	return publisherModule{
		publisher: publisher,
		logger:    logger,
	}, nil
}

type publisherModule struct {
	kernel.EssentialModule
	publisher mdlsub.Publisher
	logger    mon.Logger
}

type ExampleMessage struct {
	Greeting string `json:"greeting"`
}
```

The most interesting thing in the code above might be that we are now using a simple struct instead of having to serialize the JSON data ourselves.
We can now publish a message when our module runs.
For this we have to provide the `type` and `version` of the message as well as the body of course.
The `modelId` is automatically constructed from the data presented in the `config.dist.yml`, so we don't need to deal with that in our code.
Thus, publishing a message is done like this:

[//]: # (01_publisher_example/main.go)

```go
func (m publisherModule) Run(ctx context.Context) error {
	return m.publisher.Publish(ctx, mdlsub.TypeCreate, 0, &ExampleMessage{
		Greeting: "hello, world",
	})
}
```

There are three different types we can use.

- `mdlsub.TypeCreate` - tells a subscriber that the model was created and therefore it should insert it
- `mdlsub.TypeUpdate` - tells a subscriber that the model did already exist and needs to be updated
- `mdlsub.TypeDelete` - tells a subscriber that we removed our copy of the model, so it should do the same

This type is needed if someone subscribes to the messages we publish and uses gosoline to write them to a datastore.
The `version` has a similar use case.
It tells the subscriber the revision of the model, so if you change the data layout, you can define different transformers 
in your subscriber to handle them all correctly.

Finally, we take a look at our `config.dist.yml`.
We don't need to define a lot of settings for the publisher - only the `output_type` is strictly required.
The `modelId` for the message is constructed from `project.family.application.model`.
We can overwrite each of these fields for our publisher (for example, a `message-service-consumer` might want to publish
messages with `message-service` as the application and would therefore set the `application` property of the publisher).
If the fields are not specified, they default to `app_project`, `app_family`, `app_name`, and the name of the publisher.

[//]: # (01_publisher_example/config.dist.yml)

```yaml
env: test

app_project: gosoline
app_family: stream-example
app_name: publisher-example

mdlsub:
  publishers:
    examplePublisher: { output_type: inMemory }
```

Given this knowledge, it should be no surprise that the message we publish will look like this:

[//]: # (01_publisher_example/message.json)

```json
{
  "attributes": {
    "encoding": "application/json",
    "modelId": "gosoline.stream-example.publisher-example.examplePublisher",
    "type": "create",
    "version": 0
  },
  "body": "{\"greeting\":\"hello, world\"}"
}
```

## I want to batch messages automatically

Having less work to do to publish a single message is not the only advantage of using a `Publisher`.
For example, if you have a lot of unimportant traffic (i.e. if you lose 1% of the data, it is okay, but you still have to 
handle the data) and want to improve the performance of publishing all these messages, you might want to batch messages together.
Many systems can much easier handle few large requests than many small requests, especially if for every request a new connection
has to be established or reused from a connection pool.
It is therefore advantageous to bundle multiple messages into a larger message as well as publish multiple of these larger
messages in a single call to the service you are publishing to.

Gosoline provides you with something called the `producer daemon`.
Instead of publishing a message directly, it is forwarded to a background module which collects multiple messages and publishes
them asynchronously in the background in larger batches.
This also means that the call to `Publish` returns before the message has been sent to some external service, **and it is therefore
not guaranteed to arrive there at all**.
Should your application crash, be killed, or otherwise interrupted from publishing the batched messages **they will be lost**.
For some data this might be acceptable (the amount of lost data should be minimal, normal operations will most likely not lose 
any data at all), and the improved performance will outweigh the drawbacks.

A possible use-case for the producer daemon would be tracking user interactions with a website.
Given the sheer number of such events to expect, it would be not a big problem for some user-funnel application if you lose
a few events as long as the number is small enough to not be significant.
However, reducing the number of messages produced (e.g. written to an SQS queue) can greatly reduce your costs, so it might be worth it.

Let us therefore modify our previous example to make use of the producer daemon.
We start by changing the `Run` method to actually publish a bunch of messages:

[//]: # (02_producer_daemon_example/main.go)

```go
func (m publisherModule) Run(ctx context.Context) error {
	for i := 0; i < 10; i++ {
		err := m.publisher.Publish(ctx, mdlsub.TypeCreate, 0, &ExampleMessage{
			Greeting: fmt.Sprintf("hello from iteration %d", i),
		})

		if err != nil {
			return err
		}
	}

	return nil
}
```

That are already all the changes we need to do to the code.
To activate the producer daemon for us, we have to use the following `config.dist.yml`: 

[//]: # (02_producer_daemon_example/config.dist.yml)

```yaml
env: test

app_project: gosoline
app_family: stream-example
app_name: batch-publisher-example

mdlsub:
  publishers:
    examplePublisher: { output_type: inMemory }

stream:
  producer:
    publisher-examplePublisher:
      daemon:
        enabled: true
        aggregation_size: 5
```

So we now configure our producer to enable the producer daemon, batch up to 5 messages into a single aggregated message.
These messages get a special attribute `goso.aggregate` with the value `true` to indicate to gosoline that it needs to
recover the individual messages again when consuming such a message.
All the relevant information of a message is then encoded as json, put into an array and stored as a string in the body
of our aggregate message:

[//]: # (02_producer_daemon_example/message1.json)

```json
{
  "attributes": {
    "encoding": "application/json",
    "goso.aggregate": true,
    "modelId": "gosoline.stream-example.batch-publisher-example.examplePublisher"
  },
  "body": "[{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 0\\\"}\"},{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 1\\\"}\"},{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 2\\\"}\"},{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 3\\\"}\"},{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 4\\\"}\"}]"
}
```

The second message we produced is not much different:

[//]: # (02_producer_daemon_example/message2.json)

```json
{
  "attributes": {
    "encoding": "application/json",
    "goso.aggregate": true,
    "modelId": "gosoline.stream-example.batch-publisher-example.examplePublisher"
  },
  "body": "[{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 5\\\"}\"},{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 6\\\"}\"},{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 7\\\"}\"},{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 8\\\"}\"},{\"attributes\":{\"encoding\":\"application/json\",\"modelId\":\"gosoline.stream-example.batch-publisher-example.examplePublisher\",\"type\":\"create\",\"version\":0},\"body\":\"{\\\"greeting\\\":\\\"hello from iteration 9\\\"}\"}]"
}
```

If we would now apply compression to these messages, we could save a lot of data as we can eliminate redundancy beyond a single message.
Thus, the size per message can be much lower when aggregating the messages before compressing them compared to compressing individual messages.

Finally, there is one important thing to keep in mind about aggregated messages:
If we fail to process such a message in a consumer, we always have to acknowledge the message (if you don't acknowledge it, gosoline will do so anyway).
The reason is quite simple: When processing an aggregated message, we can only acknowledge the whole batch or messages or no message at all.
You can't acknowledge only a part of the message as that would require assembling a new message with the unacknowledged parts
and putting that back to the queue you are consuming.
Gosoline currently does not provide such a feature (and most likely never will), so if you want to retry processing aggregated messages, **you can't**.

## I don't want to write to a database by hand

TODO: Presentation of Subscriber

[//]: # (03_subscriber_example/main.go)

```go
package main

import (
	"context"
	"github.com/applike/gosoline/pkg/application"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/mdlsub"
	"github.com/applike/gosoline/pkg/stream"
	"time"
)

func main() {
	application.RunMdlSubscriber(transformers)
}

var transformers mdlsub.TransformerMapTypeVersionFactories = map[string]mdlsub.TransformerMapVersionFactories{
	"gosoline.stream-example.example.record": map[int]mdlsub.TransformerFactory{
		0: mdlsub.NewGenericTransformer(NewRecordTransformer()),
	},
}

type RecordInputV0 struct {
	Id         string    `json:"id"`
	OrderDate  time.Time `json:"orderDate"`
	CustomerId uint      `json:"customerId"`
}

type Record struct {
	Id        string    `json:"id"`
	OrderDate time.Time `json:"orderDate"`
}

func (r *Record) GetId() interface{} {
	return r.Id
}

type recordTransformer struct {
}

func NewRecordTransformer() *recordTransformer {
	go provideFakeData()

	return &recordTransformer{}
}

func provideFakeData() {
	input := stream.ProvideInMemoryInput("subscriber-record", &stream.InMemorySettings{
		Size: 3,
	})

	attributes := mdlsub.CreateMessageAttributes(mdl.ModelId{
		Project:     "gosoline",
		Family:      "stream-example",
		Application: "example",
		Name:        "record",
	}, mdlsub.TypeCreate, 0)

	// language=JSON
	msg1 := `{
		"id": "record1",
		"orderDate": "2020-02-24T12:23:00Z",
		"customerId": 15
	}`
	// language=JSON
	msg2 := `{
		"id": "record2",
		"orderDate": "2020-02-29T14:55:02Z",
		"customerId": 16
	}`
	// language=JSON
	msg3 := `{
		"id": "record3",
		"orderDate": "2020-03-12T16:07:24Z",
		"customerId": 17
	}`

	input.Publish(stream.NewJsonMessage(msg1, attributes))
	input.Publish(stream.NewJsonMessage(msg2, attributes))
	input.Publish(stream.NewJsonMessage(msg3, attributes))

	input.Stop()
}

func (r recordTransformer) GetInput() interface{} {
	return &RecordInputV0{}
}

func (r recordTransformer) Transform(_ context.Context, inp interface{}) (mdlsub.Model, error) {
	input := inp.(*RecordInputV0)

	if input.CustomerId%2 == 0 {
		return nil, nil
	}

	return &Record{
		Id:        input.Id,
		OrderDate: input.OrderDate,
	}, nil
}
```

[//]: # (03_subscriber_example/config.dist.yml)

```yaml
env: test

app_project: gosoline
app_family: stream-example
app_name: subscriber-example

kvstore:
  record:
    type: inMemory

mdlsub:
  subscribers:
    record:
      input: inMemory
      output: kvstore
      source: { application: example }
      target: { application: example }

stream:
  input:
    subscriber-record:
      type: inMemory
      size: 3
```
