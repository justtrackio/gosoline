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
	ProcessingMode      string
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

	ProcessingModeUnordered ProcessingMode = "unordered"
	ProcessingModeOrdered   ProcessingMode = "ordered"
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
	// ConsumeDelay holds every record back until it is at least this old, measured against the record's kafka
	// timestamp. Disabled by default.
	//
	// The wait happens between polling and processing a record, which also blocks rebalances because the reader
	// runs with kgo.BlockRebalanceOnPoll. Exceeding RebalanceTimeout while a rebalance is pending gets the
	// consumer kicked out of the group and leads to duplicate message processing, so the delay has to stay below
	// it. Raise rebalance_timeout if you need a longer delay.
	//
	// The record timestamp is not necessarily assigned by the broker: with the default topic setting
	// message.timestamp.type=CreateTime it comes from the producer's clock. A record whose timestamp lies in the
	// future is therefore delayed by at most ConsumeDelay, and a record without a timestamp is not delayed at all.
	ConsumeDelay time.Duration `cfg:"consume_delay" default:"0" validate:"min=0,ltfield=RebalanceTimeout"`
	// GraceTime bounds committing the offsets of an already processed batch. The commit gets this much time after
	// processing finished, so offsets still reach the broker when the consumer is shutting down.
	//
	// It does not bound processing itself: the deadline for in flight callbacks belongs to whoever owns them and is
	// passed in via exec.WithDrainContext. Under a stream consumer that is stream.consumer.<name>.grace_time; a
	// caller passing no drain context keeps full control and has its cancellation propagated immediately.
	GraceTime      time.Duration  `cfg:"grace_time"     default:"10s"`
	RunnerCount    int            `cfg:"runner_count"   default:"1" validate:"min=1"`
	ProcessingMode ProcessingMode `cfg:"processing_mode" default:"unordered" validate:"oneof=unordered ordered"`

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
