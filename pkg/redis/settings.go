package redis

import (
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/exec"
)

type Naming struct {
	Pattern string `cfg:"pattern,nodecode" default:"{name}.{group}.redis.{env}.{family}"`
}

type Settings struct {
	cfg.AppId
	Name            string `cfg:"name"`
	Dialer          string `cfg:"dialer"  default:"tcp"`
	Address         string `cfg:"address" default:"127.0.0.1:6379"`
	Naming          Naming `cfg:"naming"`
	BackoffSettings exec.BackoffSettings
}
