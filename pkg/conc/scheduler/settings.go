package scheduler

import "time"

type Settings struct {
	BatchTimeout time.Duration `cfg:"batch_timeout"  default:"10ms"`
	RunnerCount  int           `cfg:"runner_count"   default:"25"`
	MaxBatchSize int           `cfg:"max_batch_size" default:"25"`
}
