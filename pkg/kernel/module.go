package kernel

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

const (
	TypeEssential  = "essential"
	TypeForeground = "foreground"
	TypeBackground = "background"
)

func isForegroundModule(m Module) bool {
	return m.GetType() != TypeBackground
}

type ModuleState struct {
	Module    Module
	IsRunning bool
	Err       error
}

//go:generate mockery -name=Module
type Module interface {
	GetType() string
	Boot(config cfg.Config, logger mon.Logger) error
	Run(ctx context.Context) error
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
