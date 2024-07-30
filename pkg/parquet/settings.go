package parquet

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/mdl"
)

type PartitionerSettings struct {
	// at what granularity do we divide the data into partitions? Needs to be at least 1 second.
	PartitionInterval time.Duration `cfg:"partition_interval" default:"900s"  validate:"min=1000000000"`
	// how long do we buffer elements before we write them out even when the partition
	// is not yet full. Needs to be at least 1 second.
	BufferInterval time.Duration `cfg:"buffer_interval"    default:"900s"  validate:"min=1000000000"`
	// how many elements can a partition have before we have to flush it (to avoid excessive memory usage)
	MaxPartitionSize int `cfg:"max_partition_size" default:"50000" validate:"min=1"`
}

type WriterSettings struct {
	ClientName     string `cfg:"client_name" default:"default"`
	ModelId        mdl.ModelId
	NamingStrategy string
	Recorder       FileRecorder
	Tags           map[string]string
}
