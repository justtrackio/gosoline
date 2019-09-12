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

type EssentialModule struct {
}

func (m EssentialModule) GetType() string {
	return TypeEssential
}

type ForegroundModule struct {
}

func (m ForegroundModule) GetType() string {
	return TypeForeground
}

type BackgroundModule struct {
}

func (m BackgroundModule) GetType() string {
	return TypeBackground
}
