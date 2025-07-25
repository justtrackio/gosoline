package consumer

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/twmb/franz-go/pkg/kgo"
)

type (
	Balancer    string
	StartOffset string
)

const (
	FirstOffset StartOffset = "first"
	LastOffset  StartOffset = "last"

	CooperativeSticky  Balancer = "cooperative-sticky"
	Sticky             Balancer = "sticky"
	RoundRobinBalancer Balancer = "round-robin"
	Range              Balancer = "range"
)

type Settings struct {
	cfg.AppId
	Connection string `cfg:"connection" default:"default"`

	TopicId string `cfg:"topic_id" validate:"required"`
	GroupId string `cfg:"group_id" validate:"required"`

	StartOffset StartOffset `cfg:"start_offset" default:"last" validate:"oneof=first last"`
	Balancer    Balancer    `cfg:"balancer" default:"cooperative_sticky" validate:"oneof=cooperative_sticky sticky round_robin range"`

	// MaxPollRecords should not be too large as exceeding the RebalanceTimeout while still processing records
	// will get the consumer kicked out of the group and lead to duplicate message processing
	MaxPollRecords    int           `cfg:"max_poll_records" default:"100"`
	RebalanceTimeout  time.Duration `cfg:"rebalance_timeout" default:"60s"`
	SessionTimeout    time.Duration `cfg:"session_timeout" default:"45s"`
	HeartbeatInterval time.Duration `cfg:"heartbeat_interval" default:"3s"`

	Healthcheck health.HealthCheckSettings `cfg:"healthcheck"`
}

func (s *Settings) GetStartOffset() kgo.Offset {
	startOffset := kgo.NewOffset().AtStart()
	if s.StartOffset == LastOffset {
		startOffset = kgo.NewOffset().AtEnd()
	}

	return startOffset
}

func (s *Settings) GetBalancer() kgo.GroupBalancer {
	balancer := kgo.CooperativeStickyBalancer()
	switch s.Balancer {
	case Sticky:
		balancer = kgo.StickyBalancer()
	case RoundRobinBalancer:
		balancer = kgo.RoundRobinBalancer()
	case Range:
		balancer = kgo.RangeBalancer()
	}

	return balancer
}
