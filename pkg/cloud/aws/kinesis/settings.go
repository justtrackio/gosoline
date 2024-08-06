package kinesis

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsCfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoAws "github.com/justtrackio/gosoline/pkg/cloud/aws"
	"github.com/justtrackio/gosoline/pkg/exec"
)

type ClientSettings struct {
	gosoAws.ClientSettings
	ReadProvisionedThroughputDelay time.Duration `cfg:"read_provisioned_throughput_exceeded_delay" default:"1s"`
}

type ClientConfig struct {
	Settings     ClientSettings
	LoadOptions  []func(options *awsCfg.LoadOptions) error
	RetryOptions []func(*retry.StandardOptions)
}

func (c ClientConfig) GetSettings() gosoAws.ClientSettings {
	return c.Settings.ClientSettings
}

func (c ClientConfig) GetLoadOptions() []func(options *awsCfg.LoadOptions) error {
	return c.LoadOptions
}

func (c ClientConfig) GetRetryOptions() []func(*retry.StandardOptions) {
	return c.RetryOptions
}

type ClientOption func(cfg *ClientConfig)

type SettingsInitialPosition struct {
	Type      types.ShardIteratorType `cfg:"type"      default:"TRIM_HORIZON"`
	Timestamp time.Time               `cfg:"timestamp"`
}

type Settings struct {
	cfg.AppId
	// Name of the kinesis client to use
	ClientName string `cfg:"client_name"         default:"default"`
	// Name of the kinsumer
	Name string
	// Name of the stream (before expanding with project, env, family & application prefix)
	StreamName string `cfg:"stream_name"                           validate:"required"`
	// The shard reader will sleep until the age of the record is older than this delay
	ConsumeDelay time.Duration `cfg:"consume_delay"       default:"0"`
	// InitialPosition of a new kinsumer. Defines the starting position on the stream if no metadata is present.
	InitialPosition SettingsInitialPosition `cfg:"initial_position"`
	// How many records the shard reader should fetch in a single call
	MaxBatchSize int `cfg:"max_batch_size"      default:"10000"   validate:"gt=0,lte=10000"`
	// Time between reads from empty shards. This defines how fast the kinsumer begins its work. Min = 1ms
	WaitTime time.Duration `cfg:"wait_time"           default:"1s"      validate:"min=1000000"`
	// Time between writing checkpoints to ddb. This defines how much work you might lose. Min = 100ms
	PersistFrequency time.Duration `cfg:"persist_frequency"   default:"5s"      validate:"min=100000000"`
	// Time between checks for new shards. This defines how fast it reacts to shard changes. Min = 1s
	DiscoverFrequency time.Duration `cfg:"discover_frequency"  default:"1m"      validate:"min=1000000000"`
	// How long we extend the deadline of a context when releasing a shard or when deregistering a client. Min = 1s
	ReleaseDelay time.Duration `cfg:"release_delay"       default:"5s"      validate:"min=1000000000"`
	// Should we write how many milliseconds behind each shard is or only the whole stream?
	ShardLevelMetrics bool `cfg:"shard_level_metrics" default:"false"`
}

func (s Settings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s Settings) GetClientName() string {
	return s.ClientName
}

func (s Settings) GetStreamName() string {
	return s.StreamName
}

type RecordWriterSettings struct {
	ClientName string
	StreamName string
	Backoff    exec.BackoffSettings
}
