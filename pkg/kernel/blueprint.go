package kernel

import "time"

type blueprintMiddleware struct {
	factory  MiddlewareFactory
	position Position
}

type blueprintModule struct {
	name    string
	factory ModuleFactory
	options []ModuleOption
}

type blueprint struct {
	multiModuleFactories []ModuleMultiFactory
	middlewareFactories  []blueprintMiddleware
	moduleFactories      []blueprintModule
	kernelOptions        []kernelOption
}

type Option func(bp *blueprint)

func NewBlueprint(options ...Option) *blueprint {
	bp := &blueprint{}

	for _, opt := range options {
		opt(bp)
	}

	return bp
}

func WithMiddlewareFactory(factory MiddlewareFactory, position Position) Option {
	return func(bp *blueprint) {
		bp.middlewareFactories = append(bp.middlewareFactories, blueprintMiddleware{
			factory:  factory,
			position: position,
		})
	}
}

func WithModuleFactory(name string, factory ModuleFactory, options ...ModuleOption) Option {
	return func(bp *blueprint) {
		bp.moduleFactories = append(bp.moduleFactories, blueprintModule{
			name:    name,
			factory: factory,
			options: options,
		})
	}
}

func WithModuleMultiFactory(factory ModuleMultiFactory) Option {
	return func(bp *blueprint) {
		bp.multiModuleFactories = append(bp.multiModuleFactories, factory)
	}
}

func WithKillTimeout(killTimeout time.Duration) Option {
	return func(bp *blueprint) {
		bp.kernelOptions = append(bp.kernelOptions, func(k *kernel) {
			k.killTimeout = killTimeout
		})
	}
}

func WithExitHandler(handler func(code int)) Option {
	return func(bp *blueprint) {
		bp.kernelOptions = append(bp.kernelOptions, func(k *kernel) {
			k.exitHandler = handler
		})
	}
}
