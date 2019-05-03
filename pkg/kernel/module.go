package kernel

import (
	"context"
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/applike/gosoline/pkg/mon"
)

const (
	TypeForeground = "foreground"
	TypeBackground = "background"
)

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
