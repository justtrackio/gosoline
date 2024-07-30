package limit

import "time"

type FixedWindowConfig struct {
	Name   string
	Cap    int
	Window time.Duration
}
