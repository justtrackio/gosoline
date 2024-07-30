package conc

import (
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
)

type DistributedLockSettings struct {
	cfg.AppId
	Backoff         exec.BackoffSettings
	DefaultLockTime time.Duration
	Domain          string
}
