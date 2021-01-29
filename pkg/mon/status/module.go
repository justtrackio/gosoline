package status

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel"
	"github.com/applike/gosoline/pkg/mon"
	"golang.org/x/sys/unix"
	"os"
	"os/signal"
)

type module struct {
	logger        mon.Logger
	statusManager Manager
	sigChan       chan os.Signal
}

// create a new module which reports the status from the status manager upon receiving SIGUSR1
func NewModule(statusManager Manager) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger mon.Logger) (kernel.Module, error) {
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

func (m *module) GetStage() int {
	return kernel.StageService
}

func (m *module) GetType() string {
	return kernel.TypeBackground
}
