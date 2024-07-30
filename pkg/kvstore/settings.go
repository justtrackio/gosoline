package kvstore

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
)

type DdbSettings struct {
	ClientName string `cfg:"client_name" default:"default"`
}

type ChainConfiguration struct {
	Project             string                `cfg:"project"`
	Family              string                `cfg:"family"`
	Group               string                `cfg:"group"`
	Application         string                `cfg:"application"`
	Type                string                `cfg:"type"                  default:"chain" validate:"eq=chain"`
	Elements            []string              `cfg:"elements"                              validate:"min=1"`
	Ddb                 DdbSettings           `cfg:"ddb"`
	Ttl                 time.Duration         `cfg:"ttl"`
	BatchSize           int                   `cfg:"batch_size"            default:"100"   validate:"min=1"`
	MissingCacheEnabled bool                  `cfg:"missing_cache_enabled" default:"false"`
	MetricsEnabled      bool                  `cfg:"metrics_enabled"       default:"false"`
	InMemory            InMemoryConfiguration `cfg:"in_memory"`
}

type InMemoryConfiguration struct {
	MaxSize        int64  `cfg:"max_size"         default:"5000"`
	Buckets        uint32 `cfg:"buckets"          default:"16"`
	ItemsToPrune   uint32 `cfg:"items_to_prune"   default:"500"`
	DeleteBuffer   uint32 `cfg:"delete_buffer"    default:"1024"`
	PromoteBuffer  uint32 `cfg:"promote_buffer"   default:"1024"`
	GetsPerPromote int32  `cfg:"gets_per_promote" default:"3"`
}

type Settings struct {
	cfg.AppId
	DdbSettings    DdbSettings
	Name           string
	Ttl            time.Duration
	BatchSize      int
	MetricsEnabled bool
	InMemorySettings
}

type InMemorySettings struct {
	MaxSize        int64
	Buckets        uint32
	ItemsToPrune   uint32
	DeleteBuffer   uint32
	PromoteBuffer  uint32
	GetsPerPromote int32
}
