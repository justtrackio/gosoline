package kinesis

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/reslife"
	"github.com/justtrackio/gosoline/pkg/stream/health"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

type (
	Stream         string
	ClientId       string
	ShardId        string
	SequenceNumber string
	ShardIterator  string
)

type (
	shardIdSlice []ShardId
	shardInfo    struct {
		finished bool
		parent   ShardId
	}
)

type SettingsInitialPosition struct {
	Type      types.ShardIteratorType `cfg:"type" default:"TRIM_HORIZON"`
	Timestamp time.Time               `cfg:"timestamp"`
}

type Settings struct {
	cfg.AppId
	// Name of the kinesis client to use
	ClientName string `cfg:"client_name" default:"default"`
	// Name of the kinsumer
	Name string
	// Name of the stream (before expanding with project, env, family & application prefix)
	StreamName string `cfg:"stream_name" validate:"required"`
	// The shard reader will sleep until the age of the record is older than this delay
	ConsumeDelay time.Duration `cfg:"consume_delay" default:"0"`
	// InitialPosition of a new kinsumer. Defines the starting position on the stream if no metadata is present.
	InitialPosition SettingsInitialPosition `cfg:"initial_position"`
	// How many records the shard reader should fetch in a single call
	MaxBatchSize int `cfg:"max_batch_size" default:"10000" validate:"gt=0,lte=10000"`
	// Time between reads from empty or fully caught up shards. This defines how fast the kinsumer begins its work. Min = 1ms
	WaitTime time.Duration `cfg:"wait_time" default:"1s" validate:"min=1000000"`
	// Time between writing checkpoints to ddb. This defines how much work you might lose. Min = 100ms
	PersistFrequency time.Duration `cfg:"persist_frequency" default:"5s" validate:"min=100000000"`
	// How many PersistFrequency cycles do we wait until we no longer assume a client is owning a checkpoint?
	CheckpointTimeoutPeriods int `cfg:"checkpoint_timeout_periods" default:"5" validate:"min=2"`
	// Time between checks for new shards. This defines how fast it reacts to shard changes. Min = 1s
	DiscoverFrequency time.Duration `cfg:"discover_frequency" default:"15s" validate:"min=1000000000"`
	// How many DiscoverFrequency cycles do we wait until a client is considered to be gone and expired?
	ClientExpirationPeriods int `cfg:"client_expiration_periods" default:"3" validate:"min=2"`
	// How long we extend the deadline of a context when releasing a shard or when deregistering a client. Min = 1s
	ReleaseDelay time.Duration `cfg:"release_delay" default:"5s" validate:"min=1000000000"`
	// Should we ensure messages from child shards are only consumed after their parent shards have been fully consumed?
	KeepShardOrder bool `cfg:"keep_shard_order" default:"true"`
	// Healthcheck configures when we turn unhealthy and are killed
	Healthcheck health.HealthCheckSettings `cfg:"healthcheck"`
}

func (s Settings) GetAppId() cfg.AppId {
	return s.AppId
}

func (s Settings) GetClientName() string {
	return s.ClientName
}

func (s Settings) GetStreamName() string {
	return s.StreamName
}

//go:generate go run github.com/vektra/mockery/v2 --name Kinsumer
type Kinsumer interface {
	Run(ctx context.Context, handler MessageHandler) error
	Stop(ctx context.Context)
	IsHealthy() bool
}

type kinsumer struct {
	logger             log.Logger
	settings           Settings
	fullStreamName     Stream
	kinesisClient      Client
	metadataRepository MetadataRepository
	metricWriter       metric.Writer
	clock              clock.Clock
	healthCheckTimer   clock.HealthCheckTimer
	shardReaderFactory func(logger log.Logger, shardId ShardId) ShardReader
	stopLck            sync.Mutex
	stop               func()
	stopped            bool
}

type runtimeContext struct {
	clientIndex  int
	totalClients int
	shardIds     []ShardId
}

func NewKinsumer(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings) (Kinsumer, error) {
	var err error
	if err = settings.PadFromConfig(config); err != nil {
		return nil, fmt.Errorf("can not pad settings from config: %w", err)
	}
	clientId := ClientId(uuid.New().NewV4())

	var fullStreamName Stream

	if fullStreamName, err = GetStreamName(config, settings); err != nil {
		return nil, fmt.Errorf("can not get full stream name: %w", err)
	}

	logger = logger.WithChannel("kinsumer-main").WithFields(log.Fields{
		"stream_name":        fullStreamName,
		"kinsumer_client_id": clientId,
	})

	shardReaderDefaults := getShardReaderDefaultMetrics(fullStreamName)
	metricWriter := metric.NewWriter(shardReaderDefaults...)

	var kinesisClient *kinesis.Client
	var metadataRepository MetadataRepository

	if kinesisClient, err = NewClient(ctx, config, logger, settings.ClientName); err != nil {
		return nil, fmt.Errorf("failed to create kinesis client: %w", err)
	}

	if err = reslife.AddLifeCycleer(ctx, NewLifecycleManagerKinsumer(settings, clientId)); err != nil {
		return nil, fmt.Errorf("failed to add kinesis lifecycle manager: %w", err)
	}

	if metadataRepository, err = NewMetadataRepository(ctx, config, logger, fullStreamName, clientId, *settings); err != nil {
		return nil, fmt.Errorf("failed to create metadata manager: %w", err)
	}

	healthCheckTimer, err := clock.NewHealthCheckTimer(settings.Healthcheck.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to create healthcheck timer: %w", err)
	}

	shardReaderFactory := func(logger log.Logger, shardId ShardId) ShardReader {
		return NewShardReaderWithInterfaces(
			fullStreamName,
			shardId,
			logger,
			metricWriter,
			metadataRepository,
			kinesisClient,
			*settings,
			clock.Provider,
			healthCheckTimer,
		)
	}

	return NewKinsumerWithInterfaces(
		logger,
		*settings,
		fullStreamName,
		kinesisClient,
		metadataRepository,
		metricWriter,
		clock.Provider,
		healthCheckTimer,
		shardReaderFactory,
	), nil
}

func NewKinsumerWithInterfaces(
	logger log.Logger,
	settings Settings,
	fullStreamName Stream,
	kinesisClient Client,
	metadataRepository MetadataRepository,
	metricWriter metric.Writer,
	clock clock.Clock,
	healthCheckTimer clock.HealthCheckTimer,
	shardReaderFactory func(logger log.Logger, shardId ShardId) ShardReader,
) Kinsumer {
	return &kinsumer{
		logger:             logger,
		settings:           settings,
		fullStreamName:     fullStreamName,
		kinesisClient:      kinesisClient,
		metadataRepository: metadataRepository,
		metricWriter:       metricWriter,
		clock:              clock,
		healthCheckTimer:   healthCheckTimer,
		shardReaderFactory: shardReaderFactory,
	}
}

func (k *kinsumer) Run(ctx context.Context, handler MessageHandler) (finalErr error) {
	defer handler.Done()

	deregisterCtx, stop := exec.WithDelayedCancelContext(ctx, k.settings.ReleaseDelay)
	defer stop()

	// always remove the client again in the end to leave a clean client table if possible
	defer func() {
		k.logger.Info(deregisterCtx, "removing client registration")
		if err := k.metadataRepository.DeregisterClient(deregisterCtx); err != nil {
			finalErr = multierror.Append(finalErr, fmt.Errorf("failed to deregister client: %w", err))
		}
	}()

	runtimeCtx := &runtimeContext{
		clientIndex:  0,
		totalClients: 0,
		shardIds:     nil,
	}
	// don't care whether we changed, will be true anyway as we had nothing running yet
	if _, err := k.refreshShards(ctx, runtimeCtx); exec.IsRequestCanceled(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to load first list of shard ids and register as client: %w", err)
	}

	cfn, coffinCtx := coffin.WithContext(ctx)
	cancelableCoffinCtx, cancel := context.WithCancel(coffinCtx)
	k.setStop(cancel)

	cfn.GoWithContext(cancelableCoffinCtx, func(ctx context.Context) error {
		discoverTicker := k.clock.NewTicker(k.settings.DiscoverFrequency)
		defer discoverTicker.Stop()
		defer k.logger.Info(ctx, "leaving kinsumer")

		consumersWaitGroup, stopConsumers := k.startConsumers(ctx, cfn, runtimeCtx, handler)

		//nolint:gocritic // see comments below
		defer func() {
			// we need to wrap this in a function like this to ensure we call the LAST value of stopConsumers.
			// would we only do 'defer stopConsumers()', we would call the FIRST value and thus not actually cancel the
			// last set of consumers
			stopConsumers()
			// no need to wait for consumersWaitGroup here, the coffin will also wait for it to be done
		}()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-discoverTicker.Chan():
				if changed, err := k.refreshShards(ctx, runtimeCtx); exec.IsRequestCanceled(err) {
					// just terminate gracefully, if we return an error, that propagates to the top which we don't want
					return nil
				} else if err != nil {
					return fmt.Errorf("failed to refresh shards: %w", err)
				} else if !changed {
					continue
				}

				k.logger.Info(ctx, "discovered new shards or clients, restarting consumers for %d shards", len(runtimeCtx.shardIds))
				discoverTicker.Stop()
				stopConsumers()
				consumersWaitGroup.Wait()
				// Overwrite the value of stopConsumers with a new one so the above defer statement will call the correct one
				consumersWaitGroup, stopConsumers = k.startConsumers(ctx, cfn, runtimeCtx, handler)

				// reset the ticker, so we don't include the time needed to reset the consumers in the next tick
				discoverTicker.Reset(k.settings.DiscoverFrequency)
			}
		}
	})

	return cfn.Wait()
}

func (k *kinsumer) refreshShards(ctx context.Context, runtimeCtx *runtimeContext) (bool, error) {
	clientIndex, totalClients, err := k.metadataRepository.RegisterClient(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to register as client: %w", err)
	}

	k.logger.Info(ctx, "we are client %d / %d, refreshing %d shards", clientIndex+1, totalClients, len(runtimeCtx.shardIds))

	shardIds, err := k.listShardIds(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to load shards from kinesis: %w", err)
	}

	changed := totalClients != runtimeCtx.totalClients || clientIndex != runtimeCtx.clientIndex || len(runtimeCtx.shardIds) != len(shardIds)

	if !changed {
		for idx := range shardIds {
			if shardIds[idx] != runtimeCtx.shardIds[idx] {
				changed = true

				break
			}
		}
	}

	if changed {
		runtimeCtx.shardIds = shardIds
	}

	runtimeCtx.clientIndex = clientIndex
	runtimeCtx.totalClients = totalClients

	return changed, nil
}

// listShardIds returns a slice of shard ids which are not yet finished and also don't have a parent shard which is not
// yet finished (i.e., exactly those shards we need to consume next)
func (k *kinsumer) listShardIds(ctx context.Context) ([]ShardId, error) {
	shardMap := make(map[ShardId]shardInfo)
	var nextToken *string

	for {
		inputParams := kinesis.ListShardsInput{}
		if nextToken != nil {
			inputParams.NextToken = nextToken
		} else {
			inputParams.StreamName = aws.String(string(k.fullStreamName))
		}

		res, err := k.kinesisClient.ListShards(ctx, &inputParams)
		if err != nil {
			var errResourceInUseException *types.ResourceInUseException
			if errors.As(err, &errResourceInUseException) {
				return nil, NewStreamBusyError(k.fullStreamName)
			}

			var errResourceNotFoundException *types.ResourceNotFoundException
			if errors.As(err, &errResourceNotFoundException) {
				return nil, NewNoSuchStreamError(k.fullStreamName)
			}

			return nil, fmt.Errorf("failed to list shards of stream: %w", err)
		}

		for _, s := range res.Shards {
			shardId := ShardId(mdl.EmptyIfNil(s.ShardId))
			finished, err := k.metadataRepository.IsShardFinished(ctx, shardId)
			if err != nil {
				return nil, fmt.Errorf("could not check if shard is already finished: %w", err)
			}

			shardMap[shardId] = shardInfo{
				finished: finished,
				parent:   ShardId(mdl.EmptyIfNil(s.ParentShardId)),
			}
		}

		if res.NextToken == nil {
			break
		}

		nextToken = res.NextToken
	}

	shardIds := k.getSortedShardIds(shardMap)

	return shardIds, nil
}

func (k *kinsumer) getSortedShardIds(shardMap map[ShardId]shardInfo) []ShardId {
	shardIds := make([]ShardId, 0)
	for shardId, shardInfo := range shardMap {
		if shardInfo.finished {
			continue
		}

		// if a shard has a parent which no longer exists, we need to treat it like a shard without a parent (for all
		// purposes, that is true already), otherwise we can't process most shards once they have had a parent somewhere
		// in the past (but we already forgot everything about said parent)
		if _, ok := shardMap[shardInfo.parent]; !ok {
			shardInfo.parent = ""
		}

		if shardInfo.parent == "" || shardMap[shardInfo.parent].finished || !k.settings.KeepShardOrder {
			shardIds = append(shardIds, shardId)
		}
	}

	sort.Sort(shardIdSlice(shardIds))

	return shardIds
}

func (k *kinsumer) startConsumers(
	ctx context.Context,
	cfn coffin.Coffin,
	runtimeCtx *runtimeContext,
	handler MessageHandler,
) (*sync.WaitGroup, context.CancelFunc) {
	consumerCtx, stopConsumers := context.WithCancel(ctx)

	wg := &sync.WaitGroup{}
	// add one for the task writing the metrics already, so it never falls to zero while we are spawning tasks and one
	// task already finishes
	wg.Add(1)

	startedConsumers := 0

	for i := runtimeCtx.clientIndex; i < len(runtimeCtx.shardIds); i += runtimeCtx.totalClients {
		wg.Add(1)
		shardId := runtimeCtx.shardIds[i]
		logger := k.logger.WithFields(log.Fields{
			"shard_id": shardId,
		})
		startedConsumers++
		cfn.GoWithContext(consumerCtx, func(ctx context.Context) error {
			defer wg.Done()

			logger.Info(ctx, "started consuming shard")
			defer logger.Info(ctx, "done consuming shard")

			if err := k.shardReaderFactory(logger, shardId).Run(ctx, handler.Handle); err != nil {
				return fmt.Errorf("failed to consume from shard: %w", err)
			}

			return nil
		})
	}

	if startedConsumers == 0 {
		// while we have no running consumers, we must not get unhealthy, otherwise we get killed unnecessarily
		k.healthCheckTimer.MarkHealthy()
		wg.Add(1)
		// ensure we issue a tick before we get unhealthy
		ticker := k.clock.NewTicker(k.settings.Healthcheck.Timeout / 2)

		cfn.GoWithContext(consumerCtx, func(ctx context.Context) error {
			defer wg.Done()
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.Chan():
					k.healthCheckTimer.MarkHealthy()
				}
			}
		})
	}

	// we want to have one consumer / shard (ideally), so we write a metric which is above 100 if there are not enough
	// tasks running (thus, we should scale), 100, if we have exactly the correct amount, and below 100, if there
	// are too many tasks at the moment.
	// division by 0 can't happen because we are one client running, so there is at least us
	shardTaskRatio := float64(len(runtimeCtx.shardIds)) / float64(runtimeCtx.totalClients) * 100
	cfn.GoWithContext(consumerCtx, func(ctx context.Context) error {
		defer wg.Done()

		k.logger.Info(consumerCtx, "kinsumer started %d consumers for %d shards", startedConsumers, len(runtimeCtx.shardIds))
		ticker := k.clock.NewTicker(time.Minute)
		defer ticker.Stop()

		k.writeShardTaskRatioMetric(consumerCtx, shardTaskRatio)

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.Chan():
				k.writeShardTaskRatioMetric(consumerCtx, shardTaskRatio)
			}
		}
	})

	return wg, stopConsumers
}

func (k *kinsumer) writeShardTaskRatioMetric(ctx context.Context, shardTaskRatio float64) {
	// we write the shard / task ratio once for our stream (so you can track this on a per-stream basis to investigate
	// problems) and once for the whole application (taking the minimum), so if you consume two streams in one app (e.g.,
	// a subscriber), you scale to the higher number of shards of the two streams
	k.metricWriter.Write(ctx, metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameShardTaskRatioMax,
			Value:      shardTaskRatio,
			Unit:       metric.UnitCountMaximum,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameShardTaskRatio,
			Dimensions: metric.Dimensions{
				"StreamName": string(k.fullStreamName),
			},
			Value: shardTaskRatio,
			Unit:  metric.UnitCountAverage,
		},
	})
}

func (k *kinsumer) Stop(ctx context.Context) {
	var stop func()
	k.stopLck.Lock()
	stop = k.stop
	k.stop = nil
	k.stopped = true
	k.stopLck.Unlock()

	k.logger.Info(ctx, "stopping kinsumer")

	if stop != nil {
		stop()
	}
}

func (k *kinsumer) setStop(stop func()) {
	k.stopLck.Lock()
	if k.stopped {
		// call stop after unlocking the mutex again
		defer stop()
	} else {
		k.stop = stop
	}
	k.stopLck.Unlock()
}

func (k *kinsumer) IsHealthy() bool {
	return k.healthCheckTimer.IsHealthy()
}

func (s shardIdSlice) Len() int {
	return len(s)
}

func (s shardIdSlice) Less(i int, j int) bool {
	return s[i] < s[j]
}

func (s shardIdSlice) Swap(i int, j int) {
	s[i], s[j] = s[j], s[i]
}
