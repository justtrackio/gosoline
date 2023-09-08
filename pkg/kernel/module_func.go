package kernel

import "context"

type ModuleRunFunc func(ctx context.Context) error

type moduleFunc struct {
	run ModuleRunFunc
}

func NewModuleFunc(run ModuleRunFunc) Module {
	return &moduleFunc{
		run: run,
	}
}

func (m moduleFunc) Run(ctx context.Context) error {
	return m.run(ctx)
}
