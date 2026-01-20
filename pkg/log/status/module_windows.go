//go:build windows

package status

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
)

type module struct {
	kernel.BackgroundModule
	kernel.ServiceStage
}

// NewModule does nothing on Windows as SIGUSR1 is not supported
func NewModule(_ Manager) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		return &module{}, nil
	}
}

func (m *module) Run(ctx context.Context) error {
	return nil
}
