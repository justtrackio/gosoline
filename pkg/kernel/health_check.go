package kernel

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/funk"
)

type HealthCheckSettings struct {
	Timeout      time.Duration `cfg:"timeout" default:"1m"`
	WaitInterval time.Duration `cfg:"wait_interval" default:"3s"`
}

type ModuleHealthCheckResult struct {
	StageIndex int
	Name       string
	Healthy    bool
	Err        error
}

type HealthCheckResult []ModuleHealthCheckResult

func (r HealthCheckResult) GetUnhealthy() HealthCheckResult {
	return funk.Filter(r, func(result ModuleHealthCheckResult) bool {
		return !result.Healthy
	})
}

func (r HealthCheckResult) GetUnhealthyNames() []string {
	return funk.Map(r.GetUnhealthy(), func(res ModuleHealthCheckResult) string {
		return res.Name
	})
}

func (r HealthCheckResult) IsHealthy() bool {
	for _, m := range r {
		if !m.Healthy {
			return false
		}
	}

	return true
}

func (r HealthCheckResult) Err() error {
	var err error

	for _, m := range r {
		if m.Err != nil {
			err = multierror.Append(err, fmt.Errorf("error during health check in module %s: %w", m.Name, m.Err))
		}
	}

	return err
}

type (
	HealthChecker        func() HealthCheckResult
	healthCheckerKeyType int
)

var healthCheckerKey = healthCheckerKeyType(0)

func GetHealthChecker(ctx context.Context) (HealthChecker, error) {
	return appctx.Get[HealthChecker](ctx, healthCheckerKey)
}
