package consumer

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/twmb/franz-go/pkg/kgo"
)

type (
	Balancer            string
	FetchIsolationLevel string
	StartOffset         string
)

const (
	FirstOffset StartOffset = "first"
	LastOffset  StartOffset = "last"

	ReadCommitted   FetchIsolationLevel = "read_committed"
	ReadUncommitted FetchIsolationLevel = "read_uncommitted"

	CooperativeSticky  Balancer = "cooperative-sticky"
	Sticky             Balancer = "sticky"
	RoundRobinBalancer Balancer = "round-robin"
	Range              Balancer = "range"
)

type Settings struct {
	cfg.ResourceIdentifier
	Connection string `cfg:"connection" default:"default"`

	TopicId string `cfg:"topic_id" validate:"required"`
	// GroupId is an optional identifier that can be used as part of the consumer group naming pattern
	GroupId string `cfg:"group_id"`

	StartOffset         StartOffset         `cfg:"start_offset"          default:"last"               validate:"oneof=first last"`
	FetchIsolationLevel FetchIsolationLevel `cfg:"fetch_isolation_level" default:"read_uncommitted"   validate:"oneof=read_committed read_uncommitted"`
	Balancers           []Balancer          `cfg:"balancers"             default:"cooperative-sticky" validate:"dive,oneof=cooperative-sticky sticky round-robin range"`

	// MaxPollRecords should not be too large as exceeding the RebalanceTimeout while still processing records
	// will get the consumer kicked out of the group and lead to duplicate message processing
	MaxPollRecords    int           `cfg:"max_poll_records"   default:"100"`
	RebalanceTimeout  time.Duration `cfg:"rebalance_timeout"  default:"60s"`
	SessionTimeout    time.Duration `cfg:"session_timeout"    default:"45s"`
	HeartbeatInterval time.Duration `cfg:"heartbeat_interval" default:"3s"`
	IdleWaitTime      time.Duration `cfg:"idle_wait_time"     default:"500ms"`
	// ConsumeGraceTime bounds how long already-polled batches may continue processing and committing during shutdown.
	ConsumeGraceTime time.Duration `cfg:"consume_grace_time" default:"5s" validate:"min=0"`
	// ConsumeDelay delays handling a batch until the newest record has reached this age.
	// For grouped consumers, enabling it pauses fetching from the partition while waiting. franz-go
	// discards buffered records for paused partitions and refetches them after fetching resumes, which
	// removes per-partition fetch pipelining and can reduce throughput by adding broker round trips.
	ConsumeDelay time.Duration `cfg:"consume_delay" default:"0" validate:"min=0"`

	Healthcheck health.HealthCheckSettings `cfg:"healthcheck"`
	Backoff     exec.BackoffSettings       `cfg:"backoff"`
}

func (s *Settings) GetStartOffset() kgo.Offset {
	startOffset := kgo.NewOffset().AtStart()
	if s.StartOffset == LastOffset {
		startOffset = kgo.NewOffset().AtEnd()
	}

	return startOffset
}

func (s *Settings) GetFetchIsolationLevel() kgo.IsolationLevel {
	if s.FetchIsolationLevel == ReadCommitted {
		return kgo.ReadCommitted()
	}

	return kgo.ReadUncommitted()
}

func (s *Settings) GetBalancers() []kgo.GroupBalancer {
	var balancers []kgo.GroupBalancer
	for _, b := range s.Balancers {
		switch b {
		case CooperativeSticky:
			balancers = append(balancers, kgo.CooperativeStickyBalancer())
		case Sticky:
			balancers = append(balancers, kgo.StickyBalancer())
		case RoundRobinBalancer:
			balancers = append(balancers, kgo.RoundRobinBalancer())
		case Range:
			balancers = append(balancers, kgo.RangeBalancer())
		}
	}

	return balancers
}
