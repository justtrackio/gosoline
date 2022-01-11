package kinesis

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/coffin"
	"github.com/justtrackio/gosoline/pkg/exec"
	"github.com/justtrackio/gosoline/pkg/log"
	"github.com/justtrackio/gosoline/pkg/mdl"
	"github.com/justtrackio/gosoline/pkg/metric"
)

const (
	metricNameFailedRecords      = "FailedRecords"
	metricNameMillisecondsBehind = "MillisecondsBehind"
	metricNameReadCount          = "ReadCount"
	metricNameReadRecords        = "ReadRecords"
	metricNameShardTaskRatio     = "ShardTaskRatio"
)

//go:generate mockery --name ShardReader
type ShardReader interface {
	// Run reads records from this shard until we either run out of records to read (i.e., are done with the shard) or our context
	// is canceled (i.e., we should terminate, maybe, because shards got reassigned, so we need to restart all consumers)
	Run(ctx context.Context, handler func(record []byte) error) error
}

type shardReader struct {
	stream             Stream
	shardId            ShardId
	logger             log.Logger
	metricWriter       metric.Writer
	metadataRepository MetadataRepository
	kinesisClient      Client
	checkpoint         atomic.Value // Checkpoint interface, wrapped by checkpointWrapper. Stored atomically, so we can just swap it with a nop-implementation and don't need to worry about setting stuff to nil
	settings           Settings
	clock              clock.Clock
}

// checkpointWrapper is used with atomic.Value. It allows us to store different types of the same interface in it without a panic.
type checkpointWrapper struct {
	Checkpoint
}

// once we replace the checkpoint value, we use this type which doesn't care and never fails if we release it twice
type nopCheckpoint struct{}

func NewShardReaderWithInterfaces(stream Stream, shardId ShardId, logger log.Logger, metricWriter metric.Writer, metadataRepository MetadataRepository, kinesisClient Client, settings Settings, clock clock.Clock) ShardReader {
	r := &shardReader{
		stream:             stream,
		shardId:            shardId,
		logger:             logger,
		metricWriter:       metricWriter,
		metadataRepository: metadataRepository,
		kinesisClient:      kinesisClient,
		checkpoint:         atomic.Value{},
		settings:           settings,
		clock:              clock,
	}
	// store an initial nil interface, so we don't cast nil to something (which doesn't work)
	r.checkpoint.Store(checkpointWrapper{})

	return r
}

func (s *shardReader) Run(ctx context.Context, handler func(record []byte) error) (finalErr error) {
	if ok, err := s.acquireShard(ctx); errors.Is(err, ErrShardAlreadyFinished) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to acquire shard: %w", err)
	} else if !ok {
		s.logger.Info("could not acquire shard, leaving")

		return nil
	}

	s.logger.Info("acquired shard")
	defer s.logger.Info("releasing shard")

	releaseCtx := exec.WithDelayedCancelContext(ctx, s.settings.ReleaseDelay)
	defer releaseCtx.Stop()

	defer func() {
		if err := s.releaseCheckpoint(releaseCtx); err != nil {
			finalErr = multierror.Append(finalErr, fmt.Errorf("failed to release checkpoint for shard: %w", err))
		}
	}()

	sequenceNumber := s.getCheckpoint().GetSequenceNumber()
	iterator, err := s.getShardIterator(ctx, sequenceNumber)
	if err != nil {
		return fmt.Errorf("failed to get shard iterator: %w", err)
	}

	cfn, cfnCtx := coffin.WithContext(ctx)
	persisterCtx, cancelPersister := exec.WithManualCancelContext(cfnCtx)
	millisecondsBehindChan := make(chan float64)
	cfn.Go(func() error {
		// we don't use the context here because even when the context gets canceled, we need to keep draining the
		// millisecondsBehindChan until it is closed - otherwise we might block the producer on the other side
		s.reportMillisecondsBehind(millisecondsBehindChan)

		return nil
	})
	cfn.GoWithContext(persisterCtx, func(ctx context.Context) error {
		return s.runPersister(ctx, releaseCtx)
	})
	cfn.GoWithContext(cfnCtx, func(ctx context.Context) (readerErr error) {
		// similar to the outer release function, this additionally cancels the persister (and has a different error to append to)
		// and closes the channel to report how many milliseconds we lag behind
		defer func() {
			close(millisecondsBehindChan)
			cancelPersister()
			if err := s.releaseCheckpoint(releaseCtx); err != nil {
				readerErr = multierror.Append(readerErr, fmt.Errorf("failed to release checkpoint for shard: %w", err))
			}
		}()

		return s.iterateRecords(ctx, millisecondsBehindChan, iterator, handler)
	})

	return cfn.Wait()
}

func (s *shardReader) getCheckpoint() CheckpointWithoutRelease {
	return s.checkpoint.Load().(checkpointWrapper).Checkpoint
}

func (s *shardReader) releaseCheckpoint(ctx context.Context) error {
	checkpoint := s.checkpoint.Swap(checkpointWrapper{
		Checkpoint: nopCheckpoint{},
	}).(checkpointWrapper).Checkpoint

	// we need to persist the checkpoint first to ensure we propagate any changes we applied to the checkpoint before
	// releasing it
	if _, err := checkpoint.Persist(ctx); err != nil {
		return err
	}

	return checkpoint.Release(ctx)
}

func (s *shardReader) acquireShard(ctx context.Context) (bool, error) {
	for {
		checkpoint, err := s.metadataRepository.AcquireShard(ctx, s.shardId)
		if err != nil {
			return false, fmt.Errorf("failed to acquire shard: %w", err)
		}

		// store the current checkpoint
		s.checkpoint.Store(checkpointWrapper{
			Checkpoint: checkpoint,
		})

		if checkpoint != nil {
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, nil
		case <-s.clock.After(s.settings.WaitTime):
		}
	}
}

func (s *shardReader) getShardIterator(ctx context.Context, sequenceNumber SequenceNumber) (string, error) {
	input := &kinesis.GetShardIteratorInput{
		ShardId:    aws.String(string(s.shardId)),
		StreamName: aws.String(string(s.stream)),
	}

	switch sequenceNumber {
	case "":
		input.ShardIteratorType = types.ShardIteratorTypeTrimHorizon
	case "LATEST":
		input.ShardIteratorType = types.ShardIteratorTypeLatest
	default:
		input.ShardIteratorType = types.ShardIteratorTypeAfterSequenceNumber
		input.StartingSequenceNumber = aws.String(string(sequenceNumber))
	}

	resp, err := s.kinesisClient.GetShardIterator(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get shard iterator: %w", err)
	}

	return mdl.EmptyStringIfNil(resp.ShardIterator), nil
}

func (s *shardReader) getRecords(ctx context.Context, iterator string) (records []types.Record, nextIterator string, millisecondsBehind int64, err error) {
	params := &kinesis.GetRecordsInput{
		Limit:         aws.Int32(10_000), // get up to the maximum of 10 000 records
		ShardIterator: aws.String(iterator),
	}

	output, err := s.kinesisClient.GetRecords(ctx, params)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to get records from shard: %w", err)
	}

	s.writeMetric(metricNameReadCount, 1.0, metric.UnitCount)

	records = output.Records
	nextIterator = mdl.EmptyStringIfNil(output.NextShardIterator)

	return records, nextIterator, mdl.EmptyInt64IfNil(output.MillisBehindLatest), nil
}

func (s *shardReader) writeMetric(metricName string, value float64, unit metric.StandardUnit) {
	s.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricName,
		Dimensions: metric.Dimensions{
			"StreamName": string(s.stream),
		},
		Value: value,
		Unit:  unit,
	})

	if !s.settings.ShardLevelMetrics {
		return
	}

	s.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricName,
		Dimensions: metric.Dimensions{
			"StreamName": string(s.stream),
			"ShardId":    string(s.shardId),
		},
		Value: value,
		Unit:  unit,
	})
}

func (s *shardReader) runPersister(ctx context.Context, releaseCtx context.Context) error {
	persistTicker := s.clock.NewTicker(s.settings.PersistFrequency)
	defer persistTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-persistTicker.Chan():
			// if the other go routine already replaced the checkpoint, this will just call to the nop implementation,
			// so we don't need to deal with persisting and releasing twice
			shouldRelease, err := s.getCheckpoint().Persist(ctx)
			if exec.IsRequestCanceled(err) {
				// our context was canceled while writing the checkpoint, release the checkpoint we have and return.
				// the context.Canceled error is swallowed as releasing and terminating would constitute as a successful
				// execution of this task
				shouldRelease = true
			} else if err != nil {
				return fmt.Errorf("failed to persist checkpoint: %w", err)
			}

			if shouldRelease {
				if err := s.releaseCheckpoint(releaseCtx); err != nil {
					return fmt.Errorf("failed to release checkpoint for shard: %w", err)
				}

				return nil
			}
		}
	}
}

func (s *shardReader) iterateRecords(ctx context.Context, millisecondsBehindChan chan float64, iterator string, handler func(record []byte) error) error {
	timer := s.clock.NewTimer(0)
	var lastSequenceNumber SequenceNumber

	for {
		if iterator == "" {
			if err := s.getCheckpoint().Done(lastSequenceNumber); err != nil {
				return fmt.Errorf("failed to mark checkpoint as done: %w", err)
			}

			return nil
		}

		select {
		case <-ctx.Done():
			return nil
		case <-timer.Chan():
			records, nextIterator, millisecondsBehind, err := s.getRecords(ctx, iterator)
			var errExpiredIteratorException *types.ExpiredIteratorException
			if errors.As(err, &errExpiredIteratorException) {
				// we were too slow reading from the shard, so get a new iterator and continue
				iterator, err = s.getShardIterator(ctx, s.getCheckpoint().GetSequenceNumber())
				if err != nil {
					return fmt.Errorf("failed to get new shard iterator: %w", err)
				}

				timer.Reset(0)

				continue
			} else if exec.IsRequestCanceled(err) {
				// we were told to terminate while fetching new records - lets just do that. the deferred functions
				// will release the shard and shut down the persister, too.
				return nil
			} else if err != nil {
				return fmt.Errorf("failed reading records from shard: %w", err)
			}

			// send the number of ms we are behind to the metric writer task. We need to write it like that, so we
			// keep this metric correct even if we need a few minutes to process a batch. If we have a shard which we
			// are caught up to (because the shard was closed), other clients might write 0 ms behind while we write
			// nothing during processing. Thus, we need a separate task taking care of this
			millisecondsBehindChan <- float64(millisecondsBehind)

			processedSize := 0
		processRecords:
			for _, record := range records {
				if err := handler(record.Data); err != nil {
					// if we can't handle the record, we can really not do much at this point.
					// log the error and mark the record as done, returning an error would tear down the whole
					// kinsumer and retrying the record (what tearing everything down would also cause) does
					// not make sense at this point. Instead, the handler needs to implement a retry logic if needed
					s.logger.Error("failed to handle record %s: %w", record.SequenceNumber, err)

					s.writeMetric(metricNameFailedRecords, 1, metric.UnitCount)
				}

				lastSequenceNumber = SequenceNumber(mdl.EmptyStringIfNil(record.SequenceNumber))
				err = s.getCheckpoint().Advance(lastSequenceNumber)
				if err != nil {
					return fmt.Errorf("failed to advance checkpoint: %w", err)
				}

				processedSize++

				select {
				case <-ctx.Done():
					break processRecords
				default:
				}
			}

			s.logger.Info("processed batch of %d records", processedSize)
			s.writeMetric(metricNameReadRecords, float64(processedSize), metric.UnitCount)

			if len(records) > 0 || millisecondsBehind > 0 {
				timer.Reset(0)
			} else {
				timer.Reset(s.settings.WaitTime)
			}

			iterator = nextIterator
		}
	}
}

func (s *shardReader) reportMillisecondsBehind(millisecondsBehindChan chan float64) {
	// have the ticker trigger a bit faster than once a minute - otherwise we might miss a tick
	// in a minute if other work delays us getting a chance to run. We will only report the maximum
	// value to CW anyway, so it doesn't matter too much how often we run
	ticker := s.clock.NewTicker(time.Second * 15)
	defer ticker.Stop()

	currentMillisecondsBehind := 0.0
	s.writeMetric(metricNameMillisecondsBehind, currentMillisecondsBehind, metric.UnitMillisecondsMaximum)

	for {
		select {
		case <-ticker.Chan():
			s.writeMetric(metricNameMillisecondsBehind, currentMillisecondsBehind, metric.UnitMillisecondsMaximum)
		case newMillisecondsBehind, ok := <-millisecondsBehindChan:
			if !ok {
				// the producer stopped, so we also need to stop
				return
			}

			currentMillisecondsBehind = newMillisecondsBehind
			s.writeMetric(metricNameMillisecondsBehind, currentMillisecondsBehind, metric.UnitMillisecondsMaximum)
		}
	}
}

func (c nopCheckpoint) GetSequenceNumber() SequenceNumber {
	return ""
}

func (c nopCheckpoint) Advance(_ SequenceNumber) error {
	return nil
}

func (c nopCheckpoint) Done(_ SequenceNumber) error {
	return nil
}

func (c nopCheckpoint) Persist(_ context.Context) (shouldRelease bool, err error) {
	return false, nil
}

func (c nopCheckpoint) Release(_ context.Context) error {
	return nil
}

func getShardReaderDefaultMetrics(stream Stream) metric.Data {
	// no defaults for milliseconds behind - if we take a few minutes to process a batch, we are still that many ms behind
	// as we reported before - not 0 (which would be the default if we are not writing a metric). Thus, we instead leave
	// gaps in the metric to show this (thus, you maybe shouldn't define an alarm on a too short period).
	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameReadCount,
			Dimensions: map[string]string{
				"StreamName": string(stream),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameReadRecords,
			Dimensions: map[string]string{
				"StreamName": string(stream),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameFailedRecords,
			Dimensions: map[string]string{
				"StreamName": string(stream),
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
