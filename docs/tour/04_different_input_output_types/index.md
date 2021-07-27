# Different input and output types

Gosoline can read from and write to many streams or other input and output types.
This time we will take a look at these different types and how we can use them.
As the whole configuration basically happens completely in the `config.dist.yml` files, we will not look at any Go code this time.

## Input types

We already know the input type `inMemory`.
It is mainly useful for unit or integration test and only has a single setting we can configure - the size of the channel we read from (and write to in our test).
`inMemory` inputs are globally available via their name by calling `ProvideInMemoryInput` at any point.
Thus, we should not use them in production code as we don't know which other module might be messing with our input (also retrieving the input is currently not thread-safe).

The two most useful types of inputs might be `sqs` and `sns`.
Both read (despite the name of the latter) from an SQS queue, just the format of the expected message differs slightly.
You would use an `sqs` input if you have a single application knowing about the queue, for example a gateway service writing to the queue and a consumer reading from the queue.
Compared to that, an `sns` input would be the expected choice if you have some other application which doesn't really know about your application and instead just publishes messages to an SNS topic.
Your SQS queue is then subscribed to said SNS topic (hence the input name `sns`) and process the messages.
As an SNS topic basically provides a fan-out ability, you could expect other applications basically doing the same, handling the messages in different ways according to their purpose.

The most important settings for an `sqs` input are:
- `target_family` - The family part of the queue name we are reading from. Defaults to the family of your application.
- `target_application` - The application part of the queue name we are reading from. Defaults to the name of your application.
- `target_queue_id` - The identifier for the queue in the queue name. You have to specify this.

The full name of the queue is then constructed as `{family}-{env}-{application}-{queue_id}`.

The most important settings for the `sns` input are similar to the `sqs` input settings.
We have to drop the `target_` prefix (and the `queue` part) and arrive at `family`, `application`, and `id`.
They function exactly as their counterparts.

Next we have the `kinesis` input type.
The main difference of a Kinesis stream from an SQS queue is the fact that each consumer of the kinesis stream can read its own copy of a record while an SQS queue would distribute its messages among all consumers, trying to get every message just read by one consumer (if it then deletes the message).
To avoid starting to consume a stream from the beginning again every time a new instance of an application is launched, gosoline keeps track of the progress in the stream in multiple small DynamoDB tables.
They are also used to coordinate if multiple instances of your application consume the stream at the same time, so you are not bound to a single instance of your application.
To read from a stream, you need to configure the `stream_name` and the `application_name` for the input.
The `stream_name` tells gosoline which stream to consume while the `application_name` describes which application is reading the stream - each different application will get all the data from the stream separately
(so if you have multiple instances with the same `application_name` the data of the stream will be split between them).

If you use the `redis` input type, you can use gosoline to use a redis instance as a queue.
You can expect such a queue to be quite fast as it is completely held in memory, but of course it depends on your redis instance how big the queue can grow and what happens if it runs out of memory.
To use such an input, you have to provide the following config settings:
- `project` - The project the input belongs to. Defaults to the project of your application.
- `family` - The family the input belongs to. Defaults to the family of your application.
- `application` - The application the input belongs to. Defaults to the name of your application.
- `server_name` - As you can specify more than one redis server, this is used to select the correct one. If none is specified, the `default` server is used.
- `key` - Allows you to have more than one queue per server for the same project/family/application triple.
- `wait_time` - Allows you to specify how long the input blocks on each receive from redis. Defaults to `3s`.

The full name of the key accessed in redis is then constructed as `{family}-{env}-{application}-{key}`.

The last input type we will look at is `file`.
It reads a single file of newline-separated JSON documents (also called NDJSON), each document should be a `stream.Message`.
After reading the whole file it closes the input channel and stops its module.
The only setting you therefore need to provide is `filename`.
It should (to no surprise) specify the name of the newline delimited JSON file you want to process.

## Output types

For all the inputs we discussed already there is also a corresponding output.
We will mainly look at which parts differ from the inputs and are not immediately clear from the fact that we are talking about an output instead of an input now.

We already know the output type `inMemory`.
Compared to its input equivalent it has no fixed size, instead the data is collected in a slice which grows as needed.
You can use `ProvideInMemoryOutput` to retrieve a specific instance of this output type.

The output types `sqs` and `sns` will most likely be the ones you will use the most.
Their output variants are quite similar in their usage and publish a message (or multiple) to an SQS queue or SNS topic respectively.
You mainly have to configure the `project`, `family` and `application` they should use for the name of the queue or topic.
If you don't specify anything, the settings from your application will be used.
The last setting you have to specify (and this time there is no default) is the `queue_id` or `topic_id` which provide the final component of the name of the queue or topic.

For the `kinesis` output we have to specify as `stream_name` the name of the stream we want to write to.
There are no other settings we must specify for this output type.

The `redis` output needs almost the same configuration as its input.
The only setting you don't need to supply is `wait_time` (as it is not applicable in this case).
The settings `project`, `family`, `application`, `server_name` and `key` all function exactly the same as for the input we discussed already.

The last type we already know as an input is `file`.
It will produce a newline delimited JSON file specified by the `filename` setting.
You should also specify whether you want to `append` (i.e., set it to `true`) to an existing file or truncate the file (set `append` to `false`) before writing the first message.
If you append to a file, the file is assumed to either be empty or end with a newline character.
The produced file will always end with a newline character.

One quite basic output type is the `noop` output.
As the name suggest, it performs no operation for each message written to it and only discards its input.
Thus, if you for some reason need to route messages into the bit bucket once (maybe because you are writing some kind of test), `noop` would be the output to go for.

The last output type we look at has no corresponding input type (as `noop` had not, too).
It is called `multiple` and allows you to write to more than a single output at the same time.
You can configure it like this:

```yml
stream:
  output:
    myEvent:
      type: multiple
      types:
        myEventSns:
          type: sns
          # sns settings...
        myEventKinesis:
          type: kinesis
          # kinesis settings...
        # more outputs...
```

It might come in handy if you need to fan out messages to multiple data streams, but can't use e.g., SNS for that.

# Handling common errors automatically

Imagine your application is humming along happily, publishing messages to SQS, and suddenly you get back a `500 Internal Server Error` reply from SQS.
You get the error back from gosoline, return the error to your client who made the API call causing a message to get published, and now they have to deal with it.
Well, can they?
Most likely it would be much better if the client never receives the error, and we instead retry writing to the queue.
SQS is a service which may return a 500 response at any moment and this is to some part to be expected and documented.
A retry would often land at a different instance handling the request and succeed in most cases.

To avoid that you have to guard each and every call to a producer, consumer, input, output, or publisher with your own custom retry logic gosoline provides an already builtin logic.
It defaults to trying for up to 15 minutes to publish your message (with retries waiting up to 10 seconds between each call) - but it is also disabled by default.
This is the case because some APIs might be quite time sensitive (anything with a human on the other end, they don't like waiting even 10 seconds).
It is therefore your responsibility to configure the backoff an input or output should perform.

You can provide global defaults for all stream-related components like this:

```yaml
stream:
  backoff:
    enabled: false # set to true to actually enables the backoff logic
    blocking: false # if you set this to true, max_elapsed_time is set to infinity internally
    cancel_delay: 1s # how long do we try after receiving a cancel request?
    initial_interval: 50ms # how long do we wait before the first retry
    randomization_factor: 0.5 # randomness factor for random exponential backoff
    multiplier: 1.5 # by how much does the exponential backoff increase each round (before randomness is applied)
    max_interval: 10s # cap for the wait time, will never wait longer than this
    max_elapsed_time: 15m # cap for the whole operation, if we don't manage to finish the operation by then, we give up
```

For example, a small processing application reading from one queue and writing to another might simply configure it like this:

```yaml
stream:
  backoff:
    enabled: true
    blocking: true
```

Thus, any work item will cause it to only fail to process the item if the process is killed, or some unrecoverable error occurs (e.g., a malformed request is reported from SQS).
While it might look like it could stall a queue for quite some time if some item fails to be processed again and again, this normally never happens as long as SQS isn't down itself.

If you want to tune the backoff settings with more granularity, you can also overwrite the defaults per input or output:

```yaml
stream:
  backoff:
    enabled: true # enable for all inputs and outputs

  input:
    dataInput:
      type: sqs
      backoff:
        blocking: true # no need to report an error if the consumer we use with this input is stuck for some minutes, so make it blocking

  output:
    dataOutput:
      type: kinesis
      backoff:
        max_elapsed_time: 25s # the messages from our input are only not visible for 30 seconds, so we better finish writing to kinesis till then or we might process a message twice
```
