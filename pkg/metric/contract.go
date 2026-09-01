package metric

import "fmt"

// The canonical namespaces every metric gosoline emits is authored under. They are declared here
// rather than in the emitting packages because several packages emit into the same namespace: a Kafka
// consumer, a Kinesis consumer and a stream consumer all report their processing under `messaging`.
//
// The leaf of a metric is declared by the package emitting it. The conformance test in this package
// holds the full set of authored names and asserts that each one satisfies the canonical form and
// renders correctly for every writer.
const (
	// NamespaceAutoscalingPerRunner carries the per-runner values an autoscaler consumes.
	NamespaceAutoscalingPerRunner = "autoscaling.per_runner"
	// NamespaceAwsKinesisConsumer carries how a Kinesis stream's shards are distributed across clients.
	NamespaceAwsKinesisConsumer = "aws.kinesis.consumer"
	// NamespaceAwsKinesisProducer carries what a Kinesis record writer reports.
	NamespaceAwsKinesisProducer = "aws.kinesis.producer"
	// NamespaceAwsKinesisShard carries what a single Kinesis shard reader reports.
	NamespaceAwsKinesisShard = "aws.kinesis.shard"
	// NamespaceBlob carries what the blob batch runner reports.
	NamespaceBlob = "blob"
	// NamespaceConcScheduler carries what the task scheduler reports.
	NamespaceConcScheduler = "conc.scheduler"
	// NamespaceDbClient is owned by an OpenTelemetry semantic convention and carries database client
	// access, for both SQL and DynamoDB.
	NamespaceDbClient = "db.client"
	// NamespaceDbRepo carries what the SQL repository reports beyond database access itself.
	NamespaceDbRepo = "db.repo"
	// NamespaceHttpClient is owned by an OpenTelemetry semantic convention and carries outgoing HTTP
	// requests.
	NamespaceHttpClient = "http.client"
	// NamespaceHttpServer is owned by an OpenTelemetry semantic convention and carries incoming HTTP
	// requests.
	NamespaceHttpServer = "http.server"
	// NamespaceKafkaBroker carries what the Kafka client reports per broker.
	NamespaceKafkaBroker = "kafka.broker"
	// NamespaceKafkaConsumer carries what a Kafka consumer reports beyond message processing itself.
	NamespaceKafkaConsumer = "kafka.consumer"
	// NamespaceKafkaProducer carries what a Kafka producer reports beyond message sending itself.
	NamespaceKafkaProducer = "kafka.producer"
	// NamespaceKvStore carries what the key-value store reports.
	NamespaceKvStore = "kvstore"
	// NamespaceLimit carries what the rate limiter reports.
	NamespaceLimit = "limit"
	// NamespaceLog carries what the logger reports.
	NamespaceLog = "log"
	// NamespaceMdlSub carries what a model subscriber reports.
	NamespaceMdlSub = "mdlsub"
	// NamespaceMessaging is owned by an OpenTelemetry semantic convention and carries message
	// processing, consumption and production for every transport.
	NamespaceMessaging = "messaging"
	// NamespaceRpcServer is owned by an OpenTelemetry semantic convention and carries incoming gRPC
	// requests.
	NamespaceRpcServer = "rpc.server"
	// NamespaceSmpl carries what the sampler reports.
	NamespaceSmpl = "smpl"
	// NamespaceStreamConsumer carries what a stream consumer reports beyond message processing itself.
	NamespaceStreamConsumer = "stream.consumer"
	// NamespaceStreamInput carries what a stream input reports.
	NamespaceStreamInput = "stream.input"
	// NamespaceStreamInputRedisList carries what a redis list input reports.
	NamespaceStreamInputRedisList = "stream.input.redis_list"
	// NamespaceStreamOutput carries what a stream output reports.
	NamespaceStreamOutput = "stream.output"
	// NamespaceStreamOutputRedisList carries what a redis list output reports.
	NamespaceStreamOutputRedisList = "stream.output.redis_list"
	// NamespaceStreamProducer carries what the producer daemon reports.
	NamespaceStreamProducer = "stream.producer"
)

// The dimension keys shared across packages. A key an OpenTelemetry semantic convention defines is
// spelled the way the convention spells it; every other key is authored in the canonical form metric
// names use.
const (
	// DimensionErrorType is the semantic-convention attribute identifying what went wrong. It is set on
	// the metric recording an operation, so a failure does not need a metric of its own.
	DimensionErrorType = "error.type"
	// DimensionMessagingDestination is the semantic-convention attribute naming the destination a
	// message is read from or written to: a queue, a topic, a stream or a list.
	DimensionMessagingDestination = "messaging.destination.name"
	// DimensionModelId identifies the model an operation applies to.
	DimensionModelId = "model.id"
)

// ErrorType returns the value for DimensionErrorType describing an error, which is its type rather
// than its message, because a message carries unbounded cardinality. A nil error has no type.
func ErrorType(err error) string {
	if err == nil {
		return ""
	}

	return fmt.Sprintf("%T", err)
}
