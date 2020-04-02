package kernel

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/kernel/common"
	"github.com/applike/gosoline/pkg/mon"
)

const (
	TypeEssential  = common.TypeEssential
	TypeForeground = common.TypeForeground
	TypeBackground = common.TypeBackground

	// The kernel is split into three stages.
	//  * Essential: Starts first and shuts down last. Includes metric writers and anything else which gets data from other
	//    modules and must not exist before the other modules do so.
	//  * Service: Contains services provided to the application which should start first but still depend on the essential
	//    modules. This includes the modules provided by gosoline, for example the ApiServer.
	//  * Application: Your code should normally run in this stage. It will be started after other services are already
	//    running and shut down first so other stages still have some time to process the messages they did receive from
	//    your application.
	StageEssential   = common.StageEssential
	StageService     = common.StageService
	StageApplication = common.StageApplication
)

func getModuleType(m Module) string {
	if tm, ok := m.(TypedModule); ok {
		return tm.GetType()
	}

	return TypeForeground
}

func getModuleStage(m Module) int {
	if tm, ok := m.(StagedModule); ok {
		return tm.GetStage()
	}

	return StageApplication
}

func getModuleConfig(m Module) ModuleConfig {
	return ModuleConfig{
		Type:  getModuleType(m),
		Stage: getModuleStage(m),
	}
}

type ModuleState struct {
	Module    Module
	Config    ModuleConfig
	IsRunning bool
	Err       error
}

type ModuleConfig struct {
	Type  string
	Stage int
}

type ModuleOption func(ms *ModuleConfig)

// Overwrite the type a module specifies by something else.
// E.g., if you have a background module you completely depend
// on, you can do
//
// k.Add("your module", NewYourModule(), kernel.ModuleType(kernel.TypeEssential))
//
// to declare the module as essential. Now if the module quits the
// kernel will shut down instead of continuing to run.
func ModuleType(moduleType string) ModuleOption {
	return func(ms *ModuleConfig) {
		ms.Type = moduleType
	}
}

// Overwrite the stage of a module. Using this, you can move a module
// of yours (or someone else) to a different stage, e.g. to make sure it
// shuts down after another module (because it is the consumer of another
// module and you need the other module to stop producing before you can
// stop consuming).
func ModuleStage(moduleStage int) ModuleOption {
	return func(ms *ModuleConfig) {
		ms.Stage = moduleStage
	}
}

// Combine a list of options by applying them in order.
func MergeOptions(options []ModuleOption) ModuleOption {
	return func(ms *ModuleConfig) {
		for _, opt := range options {
			opt(ms)
		}
	}
}

// A module provides a single function or service for your application.
// For example, an HTTP server would be a single module (see "apiserver")
// while a daemon writing metrics in the background would be a separate
// module (see "mon").
//go:generate mockery -name=Module
type Module interface {
	// Boot the module and prepare it to run. The module is provided with
	// a logger to store (so you can write logs) and the current runtime
	// configuration.
	// If Boot returns an error, we abort kernel boot and shut down again.
	Boot(config cfg.Config, logger mon.Logger) error
	// Execute the module. If the provided context is canceled you have a
	// few seconds (configurable with kernel.killTimeout) until your module
	// is killed (via exit(1)). If you return from Run, it is assumed that
	// your module is done executing and (depending on the type of your module)
	// this might trigger a kernel shutdown. If you return an error, a kernel
	// shutdown is also triggered.
	Run(ctx context.Context) error
}

// A module can be associated with a type of TypeEssential, TypeForeground or
// TypeBackground. An essential module always causes a kernel shutdown upon
// normal termination, a foreground module only after the last foreground module
// and a background module never. If you don't implement TypedModule it will
// default to TypeForeground.
//go:generate mockery -name=TypedModule
type TypedModule interface {
	GetType() string
}

// A module can be associated with a stage describing in which order to boot and
// invoke different modules. Modules with a smaller stage index are booted sooner
// and shut down later. You should use the StageEssential, StageService and
// StageApplication constants unless you have very specific needs and know what
// you are doing.
//go:generate mockery -name=StagedModule
type StagedModule interface {
	GetStage() int
}

// A full module provides all the methods a module can have and thus never relies on defaults.
//go:generate mockery -name=FullModule
type FullModule interface {
	Module
	TypedModule
	StagedModule
}

type ModuleFactory func(config cfg.Config, logger mon.Logger) (map[string]Module, error)

// An essential module will cause the application to exit as soon as the first essential module stops running.
// For example, if you have a web server with a database and API as essential modules the application would exit
// as soon as either the database is shut down or the API is stopped. In both cases there is no point in running
// the rest anymore as the main function of the web server can no longer be fulfilled.
type EssentialModule struct {
}

func (m EssentialModule) GetType() string {
	return TypeEssential
}

// A foreground module will cause the application to exit as soon as the last foreground module exited.
// For example, if you have three tasks you have to perform and afterwards want to terminate the program,
// simply declare all three as foreground modules.
type ForegroundModule struct {
}

func (m ForegroundModule) GetType() string {
	return TypeForeground
}

// A background module has no effect on application termination. If you only have running background modules, the
// application will exit regardless.
type BackgroundModule struct {
}

func (m BackgroundModule) GetType() string {
	return TypeBackground
}

type EssentialStage struct{}

func (s EssentialStage) GetStage() int {
	return StageEssential
}

type ServiceStage struct{}

func (s ServiceStage) GetStage() int {
	return StageService
}

type ApplicationStage struct{}

func (s ApplicationStage) GetStage() int {
	return StageApplication
}

// The default module type you could use for your application code.
// Your module will
//  * Run at the application stage
//  * Be a foreground module and can therefore shut down the kernel if you don't run other foreground modules
//  * Implement any future methods we might add to the Module interface with some reasonable default values
type DefaultModule struct {
	ForegroundModule
	ApplicationStage
}
