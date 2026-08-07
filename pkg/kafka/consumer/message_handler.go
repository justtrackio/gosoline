package consumer

import "github.com/twmb/franz-go/pkg/kgo"

// BatchCompletion reports the progress of a batch of records which was handed to a KafkaMessageHandler. A
// PartitionConsumer waits for it before committing the offsets of the batch, so that we never commit a record which
// has not been processed yet.
type BatchCompletion interface {
	// Done returns a channel which gets closed once every record of the batch has been processed. Records are
	// reported as processed regardless of whether processing succeeded, as Kafka commits offsets sequentially and
	// therefore provides no way to negatively acknowledge a single record.
	Done() <-chan struct{}
	// FailedCount returns the number of records of the batch which were not processed successfully. It is only
	// meaningful once the channel returned by Done got closed.
	FailedCount() int
}

//go:generate go run github.com/vektra/mockery/v2 --name KafkaMessageHandler
type KafkaMessageHandler interface {
	// Handle passes the given records on for processing. It returns a BatchCompletion which reports once all of them
	// have been processed. Handle itself only blocks until every record has been handed over, not until it has been
	// processed.
	Handle(messages []*kgo.Record) BatchCompletion
	Stop()
}
