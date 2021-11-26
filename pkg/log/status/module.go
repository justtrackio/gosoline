package status

import (
	"context"
	"os"
	"os/signal"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"golang.org/x/sys/unix"
)

type module struct {
	kernel.BackgroundModule
	kernel.ServiceStage

	logger        log.Logger
	statusManager Manager
	sigChan       chan os.Signal
}

// NewModule creates a new module that reports the status from the status manager upon receiving SIGUSR1
func NewModule(statusManager Manager) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, unix.SIGUSR1)

		return &module{
			logger:        logger.WithChannel("status"),
			statusManager: statusManager,
			sigChan:       sigChan,
		}, nil
	}
}

func (m *module) Run(ctx context.Context) error {
	defer signal.Stop(m.sigChan)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-m.sigChan:
			m.statusManager.PrintReport(m.logger)
		}
	}
}
