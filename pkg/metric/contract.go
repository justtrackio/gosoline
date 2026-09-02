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
	// NamespaceCloudAwsKinesis carries framework-owned Kinesis metrics from pkg/cloud/aws/kinesis.
	NamespaceCloudAwsKinesis = "cloud.aws.kinesis"
	// Deprecated: use NamespaceCloudAwsKinesis. Kinesis metric namespaces no longer encode a role.
	NamespaceAwsKinesisConsumer = NamespaceCloudAwsKinesis
	// Deprecated: use NamespaceCloudAwsKinesis. Kinesis metric namespaces no longer encode a role.
	NamespaceAwsKinesisProducer = NamespaceCloudAwsKinesis
	// Deprecated: use NamespaceCloudAwsKinesis. Kinesis metric namespaces no longer encode a role.
	NamespaceAwsKinesisShard = NamespaceCloudAwsKinesis
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
	// NamespaceKafka carries framework-owned Kafka metrics from pkg/kafka.
	NamespaceKafka = "kafka"
	// Deprecated: use NamespaceKafka. Kafka metric namespaces no longer encode a role.
	NamespaceKafkaBroker = NamespaceKafka
	// NamespaceKafkaConsumer carries what a Kafka consumer reports beyond message processing itself.
	NamespaceKafkaConsumer = "kafka.consumer"
	// NamespaceKafkaProducer carries what a Kafka producer reports beyond message sending itself.
	NamespaceKafkaProducer = "kafka.producer"
	// NamespaceKvStore carries what the key-value store reports.
	NamespaceKvStore = "kvstore"
	// NamespaceLimit carries what the rate limiter reports.
	NamespaceLimit = "limit"
	// NamespaceMetric carries framework-owned metrics from pkg/metric.
	NamespaceMetric = "metric"
	// Deprecated: use NamespaceMetric. Logger metrics are emitted by pkg/metric.
	NamespaceLog = NamespaceMetric
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
	// NamespaceStream carries framework-owned metrics from pkg/stream.
	NamespaceStream = "stream"
	// Deprecated: use NamespaceStream. Stream metric namespaces no longer encode a role.
	NamespaceStreamConsumer = NamespaceStream
	// Deprecated: use NamespaceStream. Stream metric namespaces no longer encode a role.
	NamespaceStreamInput = NamespaceStream
	// Deprecated: use NamespaceStream. Stream metric namespaces no longer encode a transport.
	NamespaceStreamInputRedisList = NamespaceStream
	// Deprecated: use NamespaceStream. Stream metric namespaces no longer encode a role.
	NamespaceStreamOutput = NamespaceStream
	// Deprecated: use NamespaceStream. Stream metric namespaces no longer encode a transport.
	NamespaceStreamOutputRedisList = NamespaceStream
	// Deprecated: use NamespaceStream. Stream metric namespaces no longer encode a role.
	NamespaceStreamProducer = NamespaceStream
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
