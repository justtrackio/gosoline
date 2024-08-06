package kernel

import "time"

type HealthCheckSettings struct {
	Timeout      time.Duration `cfg:"timeout"       default:"1m"`
	WaitInterval time.Duration `cfg:"wait_interval" default:"3s"`
}

type Settings struct {
	KillTimeout time.Duration       `cfg:"kill_timeout" default:"10s"`
	HealthCheck HealthCheckSettings `cfg:"health_check"`
}
