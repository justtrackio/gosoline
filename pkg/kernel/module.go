package kernel

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/kernel/common"
	"github.com/justtrackio/gosoline/pkg/log"
)

const (
	// The kernel is split into three stages.
	//  * Essential: Starts first and shuts down last. Includes metric writers and anything else that gets data from other
	//    modules and must not exist before the other modules do so.
	//  * Service: Contains services provided to the application that should start first but still depend on the essential
	//    modules. This includes the modules provided by gosoline, for example the ApiServer.
	//  * Application: Your code should normally run in this stage. It will be started after other services are already
	//    running and shut down first so other stages still have some time to process the messages they did receive from
	//    your application.
	StageEssential   = common.StageEssential
	StageService     = common.StageService
	StageApplication = common.StageApplication
)

type (
	ModuleFactory      func(ctx context.Context, config cfg.Config, logger log.Logger) (Module, error)
	ModuleMultiFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]ModuleFactory, error)
)

func isModuleEssential(m Module) bool {
	if tm, ok := m.(TypedModule); ok {
		return tm.IsEssential()
	}

	return false
}

func isModuleBackground(m Module) bool {
	if tm, ok := m.(TypedModule); ok {
		return tm.IsBackground()
	}

	return false
}

func getModuleStage(m Module) int {
	if tm, ok := m.(StagedModule); ok {
		return tm.GetStage()
	}

	return StageApplication
}

func getModuleConfig(m Module) moduleConfig {
	return moduleConfig{
		essential:  isModuleEssential(m),
		background: isModuleBackground(m),
		stage:      getModuleStage(m),
	}
}

// moduleState stores data needed for module management.
type moduleState struct {
	// module stores the core functionality of this module.
	module Module
	// config information regarding starting and stopping this module.
	config moduleConfig
	// isRunning is 1 if the module is still running, 0 otherwise. Access with atomic reads
	isRunning int32
	// Error obtained by running this module.
	err error
}

// moduleConfig stores attributes used in starting and stopping a module.
type moduleConfig struct {
	// essential causes the kernel to stop after its first essential module finishes.
	essential bool
	// background modules run as long as there is at least one foreground module still running.
	background bool
	// stage in which this module will be started.
	stage int
}

func (mc moduleConfig) GetType() string {
	if mc.essential {
		if mc.background {
			return "essential-background"
		}

		return "essential"
	}

	if mc.background {
		return "background"
	}

	return "foreground"
}

// A Module provides a single function or service for your application.
// For example, an HTTP server would be a single module (see "httpserver")
// while a daemon writing metrics in the background would be a separate
// module (see "log").
//
//go:generate mockery --name Module
type Module interface {
	// Run the module. If the provided context is canceled you have a few seconds (configurable with kernel.killTimeout)
	// until your module is killed (via exit(1)). If you return from Run, it is assumed that your module is done
	// executing and (depending on the type of your module) this might trigger a kernel shutdown.
	// If one of your modules returns an error, the kernel will stop (thus stopping all other modules).
	Run(ctx context.Context) error
}

// TypedModule denotes a module that knows whether it is essential and whether
// it runs in the foreground or background. If your module is essential, the kernel
// will shut down after your module stopped. If your module is a background module,
// it will not keep the kernel running. Thus, an essential background module will
// kill the kernel if it stops, but it will not keep the kernel from stopping.
// An example would be the producer daemon, which should not stop, but won't do
// much good alone.
// If you don't implement TypedModule it will default to a non-essential foreground
// module.
//
//go:generate mockery --name TypedModule
type TypedModule interface {
	IsEssential() bool
	IsBackground() bool
}

var (
	_ TypedModule = BackgroundModule{}
	_ TypedModule = ForegroundModule{}
	_ TypedModule = EssentialModule{}
	_ TypedModule = EssentialBackgroundModule{}
	_ TypedModule = DefaultModule{}
)

// A module can be associated with a stage describing in which order to boot and
// invoke different modules. Modules with a smaller stage index are booted sooner
// and shut down later. You should use the StageEssential, StageService and
// StageApplication constants unless you have very specific needs and know what
// you are doing.
//
//go:generate mockery --name StagedModule
type StagedModule interface {
	GetStage() int
}

// A HealthCheckedModule provides an interface to implement a health check for a module.
// This health check is used to determine if the module is in a ready state after executing
// its run method and afterwards to check for the overall application healthyness.
//
//go:generate mockery --name HealthCheckedModule
type HealthCheckedModule interface {
	IsHealthy(ctx context.Context) (bool, error)
}

// A FullModule provides all the methods a module can have and thus never relies on defaults.
//
//go:generate mockery --name FullModule
type FullModule interface {
	Module
	TypedModule
	StagedModule
	HealthCheckedModule
}

// An EssentialModule will cause the application to exit as soon as the first essential module stops running.
// For example, if you have a web server with a database and API as essential modules the application would exit
// as soon as either the database is shut down or the API is stopped. In both cases there is no point in running
// the rest anymore as the main function of the web server can no longer be fulfilled.
type EssentialModule struct{}

func (m EssentialModule) IsEssential() bool {
	return true
}

func (m EssentialModule) IsBackground() bool {
	return false
}

// An EssentialBackgroundModule is similar to an essential module, but it will not cause the kernel to continue
// running if only this module remains. From the previous example the database might be a good candidate - the
// app can't run without the database, but a database alone also is no proper app. The ProducerDaemon is using
// this module for example.
type EssentialBackgroundModule struct{}

func (m EssentialBackgroundModule) IsEssential() bool {
	return true
}

func (m EssentialBackgroundModule) IsBackground() bool {
	return true
}

// A ForegroundModule will cause the application to exit as soon as the last foreground module exited.
// For example, if you have three tasks you have to perform and afterwards want to terminate the program,
// simply declare all three as foreground modules.
type ForegroundModule struct{}

func (m ForegroundModule) IsEssential() bool {
	return false
}

func (m ForegroundModule) IsBackground() bool {
	return false
}

// A BackgroundModule has no effect on application termination. If you only have running background modules, the
// application will exit regardless.
type BackgroundModule struct{}

func (m BackgroundModule) IsEssential() bool {
	return false
}

func (m BackgroundModule) IsBackground() bool {
	return true
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

// The DefaultModule type you could use for your application code.
// Your module will
//   - Run at the application stage
//   - Be a foreground module and can therefore shut down the kernel if you don't run other foreground modules
//   - Implement any future methods we might add to the Module interface with some reasonable default values
type DefaultModule struct {
	ForegroundModule
	ApplicationStage
}

func TypeEssential() TypedModule {
	return EssentialModule{}
}

func TypeEssentialBackground() TypedModule {
	return EssentialBackgroundModule{}
}

func TypeForeground() TypedModule {
	return ForegroundModule{}
}

func TypeBackground() TypedModule {
	return BackgroundModule{}
}
