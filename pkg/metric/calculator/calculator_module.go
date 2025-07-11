package calculator

import (
	"context"
	"fmt"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	gosoCloudwatch "github.com/justtrackio/gosoline/pkg/cloud/aws/cloudwatch"
	"github.com/justtrackio/gosoline/pkg/conc"
	"github.com/justtrackio/gosoline/pkg/conc/ddb"
	"github.com/justtrackio/gosoline/pkg/kernel"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

func CalculatorModuleFactory(ctx context.Context, config cfg.Config, logger log.Logger) (map[string]kernel.ModuleFactory, error) {
	var err error
	var handler Handler

	settings, err := readCalculatorSettings(config)
	if err != nil {
		return nil, fmt.Errorf("failed to read calculator settings: %w", err)
	}
	handlers := map[string]Handler{}
	modules := map[string]kernel.ModuleFactory{}

	if !settings.Enabled {
		return modules, nil
	}

	for name, factory := range factories {
		if handler, err = factory(ctx, config, logger, settings); err != nil {
			return nil, fmt.Errorf("failed to initialize handler %s: %w", name, err)
		}

		if handler == nil {
			continue
		}

		handlers[name] = handler
	}

	if len(handlers) == 0 {
		return modules, nil
	}

	modules["metrics-calculator"] = NewCalculatorModule(handlers, settings)

	return modules, nil
}

type CalculatorModule struct {
	kernel.EssentialModule
	kernel.ServiceStage

	logger         log.Logger
	leaderElection ddb.LeaderElection
	cwClient       gosoCloudwatch.Client
	metricWriter   metric.Writer
	ticker         clock.Ticker
	handlers       map[string]Handler
	memberId       string
	settings       *CalculatorSettings
}

func NewCalculatorModule(handlers map[string]Handler, settings *CalculatorSettings) kernel.ModuleFactory {
	return func(ctx context.Context, config cfg.Config, logger log.Logger) (kernel.Module, error) {
		logger = logger.WithChannel("metrics-calculator")

		var err error
		var leaderElection ddb.LeaderElection
		var cwClient gosoCloudwatch.Client

		if leaderElection, err = ddb.NewLeaderElection(ctx, config, logger, settings.LeaderElection); err != nil {
			return nil, fmt.Errorf("can not create leader election for metrics-per-runner module: %w", err)
		}

		if cwClient, err = gosoCloudwatch.ProvideClient(ctx, config, logger, settings.Cloudwatch.Client); err != nil {
			return nil, fmt.Errorf("can not create cloudwatch client: %w", err)
		}

		metricWriter := metric.NewWriter()
		ticker := clock.NewRealTicker(settings.Period)
		memberId := uuid.New().NewV4()

		return NewCalculatorModuleWithInterfaces(logger, leaderElection, cwClient, metricWriter, ticker, handlers, memberId, settings), nil
	}
}

func NewCalculatorModuleWithInterfaces(
	logger log.Logger,
	leaderElection ddb.LeaderElection,
	cwClient gosoCloudwatch.Client,
	metricWriter metric.Writer,
	ticker clock.Ticker,
	handlers map[string]Handler,
	memberId string,
	settings *CalculatorSettings,
) *CalculatorModule {
	return &CalculatorModule{
		logger:         logger,
		leaderElection: leaderElection,
		cwClient:       cwClient,
		metricWriter:   metricWriter,
		ticker:         ticker,
		handlers:       handlers,
		memberId:       memberId,
		settings:       settings,
	}
}

func (u *CalculatorModule) Run(ctx context.Context) error {
	for {
		if err := u.calculateMetrics(ctx); err != nil {
			return fmt.Errorf("can not write message per runner metric: %w", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-u.ticker.Chan():
			continue
		}
	}
}

func (u *CalculatorModule) calculateMetrics(ctx context.Context) error {
	var err error
	var isLeader bool
	var metrics, allMetrics metric.Data

	if isLeader, err = u.leaderElection.IsLeader(ctx, u.memberId); err != nil {
		if conc.IsLeaderElectionFatalError(err) {
			return fmt.Errorf("can not decide on leader: %w", err)
		}

		u.logger.Warn("will assume leader role as election failed: %s", err)
		isLeader = true
	}

	if !isLeader {
		u.logger.Info("not leading: do nothing")

		return nil
	}

	for name, handler := range u.handlers {
		if metrics, err = handler.GetMetrics(ctx); err != nil {
			u.logger.Warn("can not calculate metrics per runner for handler %s: %s", name, err)

			continue
		}

		allMetrics = append(allMetrics, metrics...)
	}

	u.metricWriter.Write(allMetrics)

	return nil
}
