package stream

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoKinesis "github.com/justtrackio/gosoline/pkg/cloud/aws/kinesis"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const kinsumerAutoscaleModuleName = "kinsumer-autoscale-module"

type KinsumerAutoscaleModule struct {
	kernel.BackgroundModule
	kernel.ApplicationStage

	logger            log.Logger
	kinesisClient     gosoKinesis.Client
	kinesisStreamName string
	leaderElection    ddb.LeaderElection
	memberId          string
	orchestrator      KinsumerAutoscaleOrchestrator
	settings          KinsumerAutoscaleModuleSettings
	ticker            clock.Ticker
}

func KinsumerAutoscaleModuleFactory(kinsumerInputName string) func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
		modules := map[string]kernel.ModuleFactory{}
		settings, err := readKinsumerAutoscaleSettings(config)
		if err != nil {
			return nil, fmt.Errorf("failed to read kinsumer autoscale settings in KinsumerAutoscaleModuleFactory: %w", err)
		}

		if !settings.Enabled {
			return modules, nil
		}

		// we use a static module name here to prevent that the module can be added more than once
		modules[kinsumerAutoscaleModuleName] = newKinsumerAutoscaleModule(kinsumerInputName, settings)

		return modules, nil
	}
}

func newKinsumerAutoscaleModule(kinsumerInputName string, settings KinsumerAutoscaleModuleSettings) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		logger = logger.WithChannel(kinsumerAutoscaleModuleName)

		kinsumerInputSettings, err := readKinsumerInputSettings(config, kinsumerInputName)
		if err != nil {
			return nil, fmt.Errorf("failed to read kinsumer input settings in newKinsumerAutoscaleModule: %w", err)
		}

		orchestratorFactory, ok := kinsumerAutoscaleOrchestratorFactories[settings.Orchestrator]
		if !ok {
			return nil, fmt.Errorf("invalid orchestrator of type %s", settings.Orchestrator)
		}

		orchestrator, err := orchestratorFactory(ctx, config, logger)
		if err != nil {
			return nil, fmt.Errorf("can not create orchestrator: %w", err)
		}

		kinesisClient, err := gosoKinesis.ProvideClient(ctx, config, logger, kinsumerInputSettings.ClientName)
		if err != nil {
			return nil, fmt.Errorf("can not provide kinesis client: %w", err)
		}

		kinesisStreamName, err := gosoKinesis.GetStreamName(config, kinsumerInputSettings)
		if err != nil {
			return nil, fmt.Errorf("can not get full stream name: %w", err)
		}

		leaderElection, err := ddb.NewLeaderElection(ctx, config, logger, settings.LeaderElection)
		if err != nil {
			return nil, fmt.Errorf("can not create leader election for kinsumer-autoscale-module: %w", err)
		}

		ticker := clock.NewRealTicker(settings.Period)
		memberId := uuid.New().NewV4()

		return NewKinsumerAutoscaleModuleWithInterfaces(
			logger,
			kinesisClient,
			string(kinesisStreamName),
			leaderElection,
			memberId,
			orchestrator,
			settings,
			ticker,
		), nil
	}
}

func NewKinsumerAutoscaleModuleWithInterfaces(
	logger log.Logger,
	kinesisClient gosoKinesis.Client,
	kinesisStreamName string,
	leaderElection ddb.LeaderElection,
	memberId string,
	orchestrator KinsumerAutoscaleOrchestrator,
	settings KinsumerAutoscaleModuleSettings,
	ticker clock.Ticker,
) *KinsumerAutoscaleModule {
	return &KinsumerAutoscaleModule{
		logger:            logger,
		kinesisClient:     kinesisClient,
		kinesisStreamName: kinesisStreamName,
		leaderElection:    leaderElection,
		memberId:          memberId,
		orchestrator:      orchestrator,
		settings:          settings,
		ticker:            ticker,
	}
}

func (k KinsumerAutoscaleModule) Run(ctx context.Context) error {
	for {
		if err := k.autoscaleKinsumer(ctx); err != nil {
			// ignore errors on application shutdown
			if exec.IsRequestCanceled(err) {
				return nil
			}

			return fmt.Errorf("failed to autoscale kinsumer: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-k.ticker.Chan():
			continue
		}
	}
}

func (k KinsumerAutoscaleModule) autoscaleKinsumer(ctx context.Context) error {
	logger := k.logger.WithContext(ctx)

	isLeader, err := k.leaderElection.IsLeader(ctx, k.memberId)
	if err != nil {
		if conc.IsLeaderElectionFatalError(err) {
			return fmt.Errorf("can not decide on leader: %w", err)
		}

		logger.Warn("will assume leader role as election failed: %s", err)
		isLeader = true
	}

	if !isLeader {
		logger.Info("not leading: do nothing")

		return nil
	}

	success := false
	defer func() {
		if success {
			return
		}

		if err := k.leaderElection.Resign(ctx, k.memberId); err != nil {
			logger.Warn("failed to resign leader: %s", err)
		}
	}()

	currentTaskCount, err := k.orchestrator.GetCurrentTaskCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current task count: %w", err)
	}

	shardCount, err := k.getShardCount(ctx)
	if err != nil {
		return fmt.Errorf("failed to get shard count: %w", err)
	}

	if currentTaskCount == shardCount {
		success = true

		return nil
	}

	err = k.orchestrator.UpdateTaskCount(ctx, shardCount)
	if err != nil {
		return fmt.Errorf("failed to update task count: %w", err)
	}

	success = true

	logger.Info("scaled task count from %d to %d", currentTaskCount, shardCount)

	return nil
}

func (k KinsumerAutoscaleModule) getShardCount(ctx context.Context) (int32, error) {
	describeStreamSummaryInput := &kinesis.DescribeStreamSummaryInput{
		StreamName: aws.String(k.kinesisStreamName),
	}

	describeStreamSummaryOutput, err := k.kinesisClient.DescribeStreamSummary(ctx, describeStreamSummaryInput)
	if err != nil {
		return 0, fmt.Errorf("failed to describe kinesis stream: %w", err)
	}

	shardCount := mdl.EmptyIfNil(describeStreamSummaryOutput.StreamDescriptionSummary.OpenShardCount)

	return shardCount, nil
}
