package stream

import "time"

type ProducerDaemonSettings struct {
	Enabled bool `cfg:"enabled"                default:"false"`
	// Amount of time spend waiting for messages before sending out a batch.
	Interval time.Duration `cfg:"interval"               default:"1m"`
	// Size of the buffer channel, i.e., how many messages can be in-flight at once? Generally it is a good idea to match
	// this with the number of runners.
	BufferSize int `cfg:"buffer_size"            default:"10"     validate:"min=1"`
	// Number of daemons running in the background, writing complete batches to the output.
	RunnerCount int `cfg:"runner_count"           default:"10"     validate:"min=1"`
	// How many SQS messages do we submit in a single batch? SQS can accept up to 10 messages per batch.
	// SNS doesn't support batching, so the value doesn't matter for SNS.
	BatchSize int `cfg:"batch_size"             default:"10"     validate:"min=1"`
	// How large may the sum of all messages in the aggregation be? For SQS you can't send more than 256 KB in one batch,
	// for SNS a single message can't be larger than 256 KB. We use 252 KB as default to leave some room for request
	// encoding and overhead.
	BatchMaxSize int `cfg:"batch_max_size"         default:"258048" validate:"min=0"`
	// How many stream.Messages do we pack together in a single batch (one message in SQS) at once?
	AggregationSize int `cfg:"aggregation_size"       default:"1"      validate:"min=1"`
	// Maximum size in bytes of a batch. Defaults to 64 KB to leave some room for encoding overhead.
	// Set to 0 to disable limiting the maximum size for a batch (it will still not put more than BatchSize messages
	// in a batch).
	//
	// Note: Gosoline can't ensure your messages stay below this size if your messages are quite large (especially when
	// using compression). Imagine you already aggregated 40kb of compressed messages (around 53kb when base64 encoded)
	// and are now writing a message that compresses to 20 kb. Now your buffer reaches 60 kb and 80 kb base64 encoded.
	// Gosoline will not already output a 53 kb message if you requested 64 kb messages (it would accept a 56 kb message),
	// but after writing the next message
	AggregationMaxSize int `cfg:"aggregation_max_size"   default:"65536"  validate:"min=0"`
	// If you are writing to an output using a partition key, we ensure messages are still distributed to a partition
	// according to their partition key (although not necessary the same partition as without the producer daemon).
	// For this, we split the messages into buckets while collecting them, thus potentially aggregating more messages in
	// memory (depending on the number of buckets you configure).
	//
	// Note: This still does not guarantee that your messages are perfectly ordered - this is impossible as soon as you
	// have more than once producer. However, messages with the same partition key will end up in the same shard, so if
	// you are reading two different shards and one is much further behind than the other, you will not see messages
	// *massively* out of order - it should be roughly bounded by the time you buffer messages (the Interval setting) and
	// thus be not much more than a minute (using the default setting) instead of hours (if one shard is half a day behind
	// while the other is up-to-date).
	//
	// Second note: If you change the amount of partitions, messages might move between buckets and thus end up in different
	// shards than before. Thus, only do this if you can handle it (e.g., because no shard is currently lagging behind).
	PartitionBucketCount int `cfg:"partition_bucket_count" default:"128"    validate:"min=1"`
	// Additional attributes we append to each message
	MessageAttributes map[string]string `cfg:"message_attributes"`
}
