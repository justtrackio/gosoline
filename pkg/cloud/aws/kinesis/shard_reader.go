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
	metricNameSleepDuration      = "SleepDuration"
	metricNameFailedRecords      = "FailedRecords"
	metricNameMillisecondsBehind = "MillisecondsBehind"
	metricNameProcessDuration    = "ProcessDuration"
	metricNameReadCount          = "ReadCount"
	metricNameReadRecords        = "ReadRecords"
	metricNameShardTaskRatio     = "ShardTaskRatio"
	metricNameWaitDuration       = "WaitDuration"
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

	releaseCtx, stop := exec.WithDelayedCancelContext(ctx, s.settings.ReleaseDelay)
	defer stop()

	defer func() {
		if err := s.releaseCheckpoint(releaseCtx); err != nil {
			finalErr = multierror.Append(finalErr, fmt.Errorf("failed to release checkpoint for shard: %w", err))
		}
	}()

	sequenceNumber := s.getCheckpoint().GetSequenceNumber()
	shardIterator := s.getCheckpoint().GetShardIterator()
	iterator, err := s.getShardIterator(ctx, sequenceNumber, shardIterator)
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

		return s.iterateRecords(ctx, millisecondsBehindChan, iterator, sequenceNumber, handler)
	})

	// if we get a canceled error, drop it here. We used to do this at our caller, but this makes a test harder:
	// if the context of the coffin gets canceled, we propagate the canceled error from the context to the coffin.
	// however, if already all tasks in the coffin have exited, the coffin is already dead, and we don't propagate
	// the error. Thus, it is impossible in the test to specify whether we expect the error or no error, so we just
	// clean up here.
	if err := cfn.Wait(); err != nil && !exec.IsRequestCanceled(err) {
		return err
	}

	return nil
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

func (s *shardReader) getShardIterator(ctx context.Context, sequenceNumber SequenceNumber, lastShardIterator ShardIterator) (ShardIterator, error) {
	if lastShardIterator != "" {
		// check if we can recycle the last shard iterator instead of starting from the configured starting point
		resp, err := s.kinesisClient.GetRecords(ctx, &kinesis.GetRecordsInput{
			ShardIterator: aws.String(string(lastShardIterator)),
			Limit:         aws.Int32(1),
		})
		if err != nil {
			var errExpiredIteratorException *types.ExpiredIteratorException
			if errors.As(err, &errExpiredIteratorException) {
				s.logger.WithContext(ctx).Info("can't continue from expired saved iterator, will start from configured default")
			} else {
				return "", fmt.Errorf("failed to validate shard iterator: %w", err)
			}
		} else {
			// we can actually return a fresh shard iterator and continue from there (as we didn't skip any records)
			if len(resp.Records) == 0 {
				return ShardIterator(mdl.EmptyIfNil(resp.NextShardIterator)), nil
			}

			// return our saved shard iterator and hope it doesn't expire until we request it again (shard iterators
			// expire after 5 minutes).
			return lastShardIterator, nil
		}
	}

	iteratorType := types.ShardIteratorTypeAfterSequenceNumber
	if sequenceNumber == "" {
		iteratorType = s.settings.InitialPosition.Type
	}

	input := &kinesis.GetShardIteratorInput{
		ShardId:           aws.String(string(s.shardId)),
		StreamName:        aws.String(string(s.stream)),
		ShardIteratorType: iteratorType,
	}

	switch iteratorType {
	case types.ShardIteratorTypeAtTimestamp:
		input.Timestamp = mdl.Box(s.settings.InitialPosition.Timestamp)
	case types.ShardIteratorTypeAfterSequenceNumber:
		input.StartingSequenceNumber = aws.String(string(sequenceNumber))
	}

	resp, err := s.kinesisClient.GetShardIterator(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get shard iterator: %w", err)
	}

	return ShardIterator(mdl.EmptyIfNil(resp.ShardIterator)), nil
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

func (s *shardReader) iterateRecords(ctx context.Context, millisecondsBehindChan chan float64, iterator ShardIterator, startingSequenceNumber SequenceNumber, handler func(record []byte) error) error {
	timer := s.clock.NewTimer(0)
	// we have to carry the old sequence number forward - otherwise we could have the following scenario:
	// - our stream is empty for more than one day (if data expires after one day)
	// - our service is redeployed, we have no longer a sequence number (if we wouldn't start with the last one)
	// - later, our service is stopped for a few minutes, causing the shard iterator we store to expire
	// - during these minutes, some data is written to the stream
	// - we now start without a sequence number nor a valid iterator and start from the configured default - which
	//   might be something else than TRIM_HORIZON (for example, in case of LATEST we would lose some data)
	lastSequenceNumber := startingSequenceNumber

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
			getRecordsStart := s.clock.Now()
			records, nextIterator, millisecondsBehind, err := s.getRecords(ctx, iterator)

			var errExpiredIteratorException *types.ExpiredIteratorException
			if errors.As(err, &errExpiredIteratorException) {
				// we were too slow reading from the shard, so get a new iterator and continue
				sequenceNumber := s.getCheckpoint().GetSequenceNumber()
				shardIterator := s.getCheckpoint().GetShardIterator()
				iterator, err = s.getShardIterator(ctx, sequenceNumber, shardIterator)
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

			processStart := s.clock.Now()
			var processedSize int

			if processedSize, err = s.processRecords(ctx, records, &lastSequenceNumber, iterator, handler); err != nil {
				return err
			} else if processedSize == len(records) {
				// only advance the iterator if we processed the whole batch - if we don't do it like this, we could get
				// canceled while processing a batch, but actually process the last iterator of the shard (so nextIterator is "")
				// and thus would mark the shard as finished - losing the last few records from that shard
				iterator = nextIterator
			}

			processDuration := s.clock.Since(processStart)
			s.writeMetric(metricNameProcessDuration, float64(processDuration.Milliseconds()), metric.UnitMillisecondsAverage)
			s.writeMetric(metricNameReadRecords, float64(processedSize), metric.UnitCount)

			s.logger.WithChannel("kinsumer-read").WithFields(log.Fields{
				"count":       processedSize,
				"duration_ms": processDuration.Milliseconds(),
			}).Info("processed batch of %d records in %s", processedSize, processDuration)

			// if the results are older than our wait time, continue immediately
			if time.Duration(millisecondsBehind) > (s.settings.WaitTime + s.settings.ConsumeDelay) {
				s.writeMetric(metricNameWaitDuration, 0.0, metric.UnitMillisecondsAverage)
				timer.Reset(0)
				continue
			}

			durationSinceLastGetRecordsCall := s.clock.Since(getRecordsStart)
			waitTime := s.settings.WaitTime - durationSinceLastGetRecordsCall

			if waitTime < 0 {
				waitTime = 0
			}

			s.writeMetric(metricNameWaitDuration, float64(waitTime.Milliseconds()), metric.UnitMillisecondsAverage)
			timer.Reset(waitTime)
		}
	}
}

func (s *shardReader) getRecords(ctx context.Context, iterator ShardIterator) (records []types.Record, nextIterator ShardIterator, millisecondsBehind int64, err error) {
	params := &kinesis.GetRecordsInput{
		Limit:         aws.Int32(int32(s.settings.MaxBatchSize)),
		ShardIterator: aws.String(string(iterator)),
	}

	output, err := s.kinesisClient.GetRecords(ctx, params)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to get records from shard: %w", err)
	}

	s.writeMetric(metricNameReadCount, 1.0, metric.UnitCount)

	records = output.Records
	nextIterator = ShardIterator(mdl.EmptyIfNil(output.NextShardIterator))

	return records, nextIterator, mdl.EmptyIfNil(output.MillisBehindLatest), nil
}

func (s *shardReader) processRecords(ctx context.Context, records []types.Record, lastSequenceNumber *SequenceNumber, nextIterator ShardIterator, handler func(record []byte) error) (int, error) {
	processedSize := 0

	// if our batch is empty, just write the next iterator to the checkpoint
	if len(records) == 0 {
		err := s.getCheckpoint().Advance(*lastSequenceNumber, nextIterator)
		if err != nil {
			return processedSize, fmt.Errorf("failed to advance checkpoint: %w", err)
		}
	}

	for _, record := range records {
		s.delayConsume(ctx, record)

		// if context was canceled in the meantime, stop processing
		select {
		case <-ctx.Done():
			return processedSize, nil
		default:
		}

		if err := handler(record.Data); err != nil {
			// if we can't handle the record, we can really not do much at this point.
			// log the error and mark the record as done, returning an error would tear down the whole
			// kinsumer and retrying the record (what tearing everything down would also cause) does
			// not make sense at this point. Instead, the handler needs to implement a retry logic if needed
			s.logger.Error("failed to handle record %s: %w", record.SequenceNumber, err)

			s.writeMetric(metricNameFailedRecords, 1, metric.UnitCount)
		}

		*lastSequenceNumber = SequenceNumber(mdl.EmptyIfNil(record.SequenceNumber))
		// if we process any record, we don't need to store a shard iterator (they are only valid for 5 minutes, so if we
		// processed a record, the stream is not empty, and we should consume the next batch before all the records in the
		// stream expire (which is at least a day away if we are keeping up with the stream)).
		err := s.getCheckpoint().Advance(*lastSequenceNumber, "")
		if err != nil {
			return processedSize, fmt.Errorf("failed to advance checkpoint: %w", err)
		}

		processedSize++

		// if context was canceled in the meantime, stop processing
		select {
		case <-ctx.Done():
			return processedSize, nil
		default:
		}
	}

	return processedSize, nil
}

func (s *shardReader) delayConsume(ctx context.Context, record types.Record) {
	// don't sleep if we don't want to
	if s.settings.ConsumeDelay == 0 {
		return
	}

	now := s.clock.Now()
	recordAge := now.Sub(*record.ApproximateArrivalTimestamp)

	// don't sleep if the record is already older than the delay
	if recordAge >= s.settings.ConsumeDelay {
		return
	}

	durationToSleep := s.settings.ConsumeDelay - recordAge
	timer := s.clock.NewTimer(durationToSleep)
	defer timer.Stop()

	select {
	case <-ctx.Done():
	case <-timer.Chan():
		s.writeMetric(metricNameSleepDuration, float64(durationToSleep.Milliseconds()), metric.UnitMillisecondsAverage)
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
