package stream

import (
	"context"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/log"
)

const orchestratorTypeEcs = "ecs"

//go:generate mockery --name KinsumerAutoscaleOrchestrator
type KinsumerAutoscaleOrchestrator interface {
	GetCurrentTaskCount(ctx context.Context) (int32, error)
	UpdateTaskCount(ctx context.Context, taskCount int32) error
}

type KinsumerAutoscaleOrchestratorFactory func(ctx context.Context, config cfg.Config, logger log.Logger) (KinsumerAutoscaleOrchestrator, error)

var kinsumerAutoscaleOrchestratorFactories = map[string]KinsumerAutoscaleOrchestratorFactory{
	orchestratorTypeEcs: newKinsumerAutoscaleOrchestratorECS,
}
