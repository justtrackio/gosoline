package stream

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/justtrackio/gosoline/pkg/cfg"
	gosoEcs "github.com/justtrackio/gosoline/pkg/cloud/aws/ecs"
	"github.com/justtrackio/gosoline/pkg/log"
)

type kinsumerAutoscaleOrchestratorECS struct {
	ecsClient gosoEcs.Client
	settings  KinsumerAutoscaleModuleSettings
}

func newKinsumerAutoscaleOrchestratorECS(ctx context.Context, config cfg.Config, logger log.Logger) (KinsumerAutoscaleOrchestrator, error) {
	settings, err := readKinsumerAutoscaleSettings(config)
	if err != nil {
		return nil, fmt.Errorf("can not read kinsumer autoscale settings: %w", err)
	}

	ecsClient, err := gosoEcs.ProvideClient(ctx, config, logger, settings.Ecs.Client)
	if err != nil {
		return nil, fmt.Errorf("can not provide ecs client: %w", err)
	}

	return &kinsumerAutoscaleOrchestratorECS{
		ecsClient: ecsClient,
		settings:  settings,
	}, nil
}

func (k kinsumerAutoscaleOrchestratorECS) GetCurrentTaskCount(ctx context.Context) (int32, error) {
	describeServicesInput := &ecs.DescribeServicesInput{
		Services: []string{k.settings.Ecs.Service},
		Cluster:  aws.String(k.settings.Ecs.Cluster),
	}

	servicesOutput, err := k.ecsClient.DescribeServices(ctx, describeServicesInput)
	if err != nil {
		return 0, fmt.Errorf("failed to describe ecs service: %w", err)
	}

	currentDesiredCount := servicesOutput.Services[0].DesiredCount

	return currentDesiredCount, nil
}

func (k kinsumerAutoscaleOrchestratorECS) UpdateTaskCount(ctx context.Context, taskCount int32) error {
	updateServiceInput := &ecs.UpdateServiceInput{
		Service:      aws.String(k.settings.Ecs.Service),
		Cluster:      aws.String(k.settings.Ecs.Cluster),
		DesiredCount: aws.Int32(taskCount),
	}

	_, err := k.ecsClient.UpdateService(ctx, updateServiceInput)
	if err != nil {
		return fmt.Errorf("failed to update ecs service: %w", err)
	}

	return nil
}
