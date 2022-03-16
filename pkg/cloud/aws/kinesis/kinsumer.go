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
	"github.com/justtrackio/gosoline/pkg/appctx"
	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/uuid"
)

const (
	MetadataKeyKinsumers = "cloud.aws.kinesis.kinsumers"
)

type KinsumerMetadata struct {
	ClientId       ClientId  `json:"client_id"`
	Name           string    `json:"name"`
	StreamAppId    cfg.AppId `json:"stream_app_id"`
	StreamName     string    `json:"stream_name"`
	StreamNameFull Stream    `json:"stream_name_full"`
}

type (
	Stream         string
	ClientId       string
	ShardId        string
	SequenceNumber string
)

type shardIdSlice []ShardId

type SettingsInitialPosition struct {
	Type      types.ShardIteratorType `cfg:"type" default:"TRIM_HORIZON"`
	Timestamp time.Time               `cfg:"timestamp"`
}

type Settings struct {
	cfg.AppId
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
	// Time between reads from empty shards. This defines how fast the kinsumer begins its work. Min = 1ms
	WaitTime time.Duration `cfg:"wait_time" default:"1s" validate:"min=1000000"`
	// Time between writing checkpoints to ddb. This defines how much work you might lose. Min = 100ms
	PersistFrequency time.Duration `cfg:"persist_frequency" default:"5s" validate:"min=100000000"`
	// Time between checks for new shards. This defines how fast it reacts to shard changes. Min = 1s
	DiscoverFrequency time.Duration `cfg:"discover_frequency" default:"1m" validate:"min=1000000000"`
	// How long we extend the deadline of a context when releasing a shard or when deregistering a client. Min = 1s
	ReleaseDelay time.Duration `cfg:"release_delay" default:"5s" validate:"min=1000000000"`
	// Should we write how many milliseconds behind each shard is or only the whole stream?
	ShardLevelMetrics bool `cfg:"shard_level_metrics" default:"false"`
}

//go:generate mockery --name Kinsumer
type Kinsumer interface {
	Run(ctx context.Context, handler MessageHandler) error
	Stop()
}

type kinsumer struct {
	logger             log.Logger
	settings           Settings
	stream             Stream
	kinesisClient      Client
	metadataRepository MetadataRepository
	metricWriter       metric.Writer
	clock              clock.Clock
	shardReaderFactory func(logger log.Logger, shardId ShardId) ShardReader
	stop               func()
}

type runtimeContext struct {
	clientIndex  int
	totalClients int
	shardIds     []ShardId
}

func NewKinsumer(ctx context.Context, config cfg.Config, logger log.Logger, settings *Settings) (Kinsumer, error) {
	settings.PadFromConfig(config)
	fullStreamName := Stream(fmt.Sprintf("%s-%s-%s-%s-%s", settings.Project, settings.Environment, settings.Family, settings.Application, settings.StreamName))
	clientId := ClientId(uuid.New().NewV4())

	logger = logger.WithChannel("kinsumer-main").WithFields(log.Fields{
		"stream_name":        fullStreamName,
		"kinsumer_client_id": clientId,
	})

	shardReaderDefaults := getShardReaderDefaultMetrics(fullStreamName)
	metricWriter := metric.NewDaemonWriter(shardReaderDefaults...)

	var err error
	var kinesisClient *kinesis.Client
	var metadataRepository MetadataRepository

	if kinesisClient, err = NewClient(ctx, config, logger, "default"); err != nil {
		return nil, fmt.Errorf("failed to create kinesis client: %w", err)
	}

	if err = CreateKinesisStream(ctx, config, logger, kinesisClient, string(fullStreamName)); err != nil {
		return nil, fmt.Errorf("failed to create kinesis stream: %w", err)
	}

	if metadataRepository, err = NewMetadataRepository(ctx, config, logger, fullStreamName, clientId, *settings); err != nil {
		return nil, fmt.Errorf("failed to create metadata manager: %w", err)
	}

	kinsumerMetadata := KinsumerMetadata{
		ClientId:       clientId,
		Name:           settings.Name,
		StreamAppId:    settings.AppId,
		StreamName:     settings.StreamName,
		StreamNameFull: fullStreamName,
	}
	if err = appctx.MetadataAppend(ctx, MetadataKeyKinsumers, kinsumerMetadata); err != nil {
		return nil, fmt.Errorf("can not access the appctx metadata: %w", err)
	}

	shardReaderFactory := func(logger log.Logger, shardId ShardId) ShardReader {
		return NewShardReaderWithInterfaces(fullStreamName, shardId, logger, metricWriter, metadataRepository, kinesisClient, *settings, clock.Provider)
	}

	return NewKinsumerWithInterfaces(logger, *settings, fullStreamName, kinesisClient, metadataRepository, metricWriter, clock.Provider, shardReaderFactory), nil
}

func NewKinsumerWithInterfaces(logger log.Logger, settings Settings, stream Stream, kinesisClient Client, metadataRepository MetadataRepository, metricWriter metric.Writer, clock clock.Clock, shardReaderFactory func(logger log.Logger, shardId ShardId) ShardReader) Kinsumer {
	return &kinsumer{
		logger:             logger,
		settings:           settings,
		stream:             stream,
		kinesisClient:      kinesisClient,
		metadataRepository: metadataRepository,
		metricWriter:       metricWriter,
		clock:              clock,
		shardReaderFactory: shardReaderFactory,
	}
}

func (k *kinsumer) Run(ctx context.Context, handler MessageHandler) (finalErr error) {
	deregisterCtx, stop := exec.WithDelayedCancelContext(ctx, k.settings.ReleaseDelay)
	defer stop()

	logger := k.logger.WithContext(ctx)

	// always remove the client again in the end to leave a clean client table if possible
	defer func() {
		logger.Info("removing client registration")
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
	if _, err := k.refreshShards(ctx, runtimeCtx); err != nil {
		return fmt.Errorf("failed to load first list of shard ids and register as client: %w", err)
	}

	cfn, coffinCtx := coffin.WithContext(ctx)
	cancelableCoffinCtx, cancel := context.WithCancel(coffinCtx)
	k.stop = cancel

	cfn.GoWithContext(cancelableCoffinCtx, func(ctx context.Context) error {
		discoverTicker := k.clock.NewTicker(k.settings.DiscoverFrequency)
		defer discoverTicker.Stop()
		defer logger.Info("leaving kinsumer")

		consumersWaitGroup, stopConsumers := k.startConsumers(ctx, cfn, runtimeCtx, handler)
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

				logger.Info("discovered new shards or clients, restarting consumers for %d shards", len(runtimeCtx.shardIds))
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

	defer handler.Done()

	return cfn.Wait()
}

func (k *kinsumer) refreshShards(ctx context.Context, runtimeCtx *runtimeContext) (bool, error) {
	logger := k.logger.WithContext(ctx)

	clientIndex, totalClients, err := k.metadataRepository.RegisterClient(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to register as client: %w", err)
	}

	logger.Info("we are client %d / %d, refreshing %d shards", clientIndex+1, totalClients, len(runtimeCtx.shardIds))

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
	type shardInfo struct {
		finished bool
		parent   ShardId
	}

	shardMap := make(map[ShardId]shardInfo)
	var nextToken *string

	for {
		inputParams := kinesis.ListShardsInput{}
		if nextToken != nil {
			inputParams.NextToken = nextToken
		} else {
			inputParams.StreamName = aws.String(string(k.stream))
		}

		res, err := k.kinesisClient.ListShards(ctx, &inputParams)
		if err != nil {
			var errResourceInUseException *types.ResourceInUseException
			if errors.As(err, &errResourceInUseException) {
				return nil, NewStreamBusyError(k.stream)
			}

			var errResourceNotFoundException *types.ResourceNotFoundException
			if errors.As(err, &errResourceNotFoundException) {
				return nil, NewNoSuchStreamError(k.stream)
			}

			return nil, fmt.Errorf("failed to list shards of stream: %w", err)
		}

		for _, s := range res.Shards {
			shardId := ShardId(mdl.EmptyStringIfNil(s.ShardId))
			finished, err := k.metadataRepository.IsShardFinished(ctx, shardId)
			if err != nil {
				return nil, fmt.Errorf("could not check if shard is already finished: %w", err)
			}

			shardMap[shardId] = shardInfo{
				finished: finished,
				parent:   ShardId(mdl.EmptyStringIfNil(s.ParentShardId)),
			}
		}

		if res.NextToken == nil {
			break
		}

		nextToken = res.NextToken
	}

	shardIds := make([]ShardId, 0)
	for k, v := range shardMap {
		if v.finished {
			continue
		}

		// if a shard has a parent which no longer exists, we need to treat it like a shard without a parent (for all
		// purposes, that is true already), otherwise we can't process most shards once they have had a parent somewhere
		// in the past (but we already forgot everything about said parent)
		if _, ok := shardMap[v.parent]; !ok {
			v.parent = ""
		}

		if v.parent == "" || shardMap[v.parent].finished {
			shardIds = append(shardIds, k)
		}
	}

	sort.Sort(shardIdSlice(shardIds))

	return shardIds, nil
}

func (k *kinsumer) startConsumers(ctx context.Context, cfn coffin.Coffin, runtimeCtx *runtimeContext, handler MessageHandler) (*sync.WaitGroup, context.CancelFunc) {
	consumerCtx, stopConsumers := context.WithCancel(ctx)

	wg := &sync.WaitGroup{}
	// add one for the task writing the metrics already, so it never falls to zero while we are spawning tasks and one
	// task already finishes
	wg.Add(1)

	logger := k.logger.WithContext(consumerCtx)
	startedConsumers := 0

	for i := runtimeCtx.clientIndex; i < len(runtimeCtx.shardIds); i += runtimeCtx.totalClients {
		wg.Add(1)
		shardId := runtimeCtx.shardIds[i]
		logger := logger.WithFields(log.Fields{
			"shard_id": shardId,
		})
		startedConsumers++
		cfn.GoWithContext(consumerCtx, func(ctx context.Context) error {
			defer wg.Done()

			logger.Info("started consuming shard")
			defer logger.Info("done consuming shard")

			if err := k.shardReaderFactory(logger, shardId).Run(ctx, handler.Handle); err != nil {
				return fmt.Errorf("failed to consume from shard: %w", err)
			}

			return nil
		})
	}

	// we want to have one consumer / shard (ideally), so we write a metric which is above 100 if there are not enough
	// tasks running (thus, we should scale), 100, if we have exactly the correct amount, and below 100, if there
	// are too many tasks at the moment.
	// division by 0 can't happen because we are one client running, so there is at least us
	shardTaskRatio := float64(len(runtimeCtx.shardIds)) / float64(runtimeCtx.totalClients) * 100
	cfn.GoWithContext(consumerCtx, func(ctx context.Context) error {
		defer wg.Done()

		logger.Info("kinsumer started %d consumers for %d shards", startedConsumers, len(runtimeCtx.shardIds))
		ticker := k.clock.NewTicker(time.Minute)
		defer ticker.Stop()

		k.writeShardTaskRatioMetric(shardTaskRatio)

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.Chan():
				k.writeShardTaskRatioMetric(shardTaskRatio)
			}
		}
	})

	return wg, stopConsumers
}

func (k *kinsumer) writeShardTaskRatioMetric(shardTaskRatio float64) {
	// we write the shard / task ratio once for our stream (so you can track this on a per-stream basis to investigate
	// problems) and once for the whole application (taking the minimum), so if you consume two streams in one app (e.g.,
	// a subscriber), you scale to the higher number of shards of the two streams
	k.metricWriter.Write(metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameShardTaskRatio,
			Value:      shardTaskRatio,
			Unit:       metric.UnitCountMaximum,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameShardTaskRatio,
			Dimensions: metric.Dimensions{
				"StreamName": string(k.stream),
			},
			Value: shardTaskRatio,
			Unit:  metric.UnitCountAverage,
		},
	})
}

func (k *kinsumer) Stop() {
	if k.stop != nil {
		k.logger.Info("stopping kinsumer")
		k.stop()
	}
	k.stop = nil
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
