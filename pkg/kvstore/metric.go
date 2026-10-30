package kvstore

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/justtrackio/gosoline/pkg/metric"
	"github.com/justtrackio/gosoline/pkg/refl"
)

const (
	metricNamespace = "kvstore"

	// number of items currently stored in the store (if available)
	metricNameKvStoreItemCount = "item.count"
	// number of successful read items, separated by whether the store served them itself
	metricNameKvStoreReads = "reads"
	// number of read items requested from the store, including failed calls
	metricNameKvStoreAttemptedReads = "attempted.reads"
	// number of items written to the store
	metricNameKvStoreWrites = "writes"
	// number of items deleted from the store
	metricNameKvStoreDeletes = "deletes"

	dimensionModelId   = "model.id"
	dimensionStoreType = "store.type"

	// dimensionHit tells apart a read the store served from its own data from one it did not.
	dimensionHit = "hit"
)

func init() {
	metric.RegisterHelp(metricNamespace, metricNameKvStoreReads, "successful read operations performed against a key-value store, by whether the store served them itself")
	metric.RegisterHelp(metricNamespace, metricNameKvStoreAttemptedReads, "read operations attempted against a key-value store")
	metric.RegisterHelp(metricNamespace, metricNameKvStoreWrites, "write operations performed against a key-value store")
	metric.RegisterHelp(metricNamespace, metricNameKvStoreDeletes, "delete operations performed against a key-value store")
	metric.RegisterHelp(metricNamespace, metricNameKvStoreItemCount, "items a key-value store currently holds")
}

type MetricStore[T any] struct {
	KvStore[T]
	metricWriter metric.Writer
	model        string
	store        string
}

func NewMetricStoreWithInterfaces[T any](store KvStore[T], settings *Settings) KvStore[T] {
	if !settings.MetricsEnabled {
		return store
	}

	modelIdString := settings.String()
	storeName := fmt.Sprintf("%T", store)
	defaults := getDefaultMetrics(modelIdString, storeName)

	s := &MetricStore[T]{
		KvStore:      store,
		metricWriter: metric.NewWriter(metricNamespace, defaults...),
		model:        modelIdString,
		store:        storeName,
	}

	if sizedStore, ok := store.(SizedStore[T]); ok {
		go s.recordSize(sizedStore)
	}

	return s
}

func (s *MetricStore[T]) Contains(ctx context.Context, key any) (bool, error) {
	s.recordAttemptedReads(ctx, 1)
	found, err := s.KvStore.Contains(ctx, key)

	if err == nil {
		s.recordReads(ctx, 1, boolToInt(found))
	}

	return found, err
}

func (s *MetricStore[T]) Get(ctx context.Context, key any, value *T) (bool, error) {
	s.recordAttemptedReads(ctx, 1)
	found, err := s.KvStore.Get(ctx, key, value)

	if err == nil {
		s.recordReads(ctx, 1, boolToInt(found))
	}

	return found, err
}

func (s *MetricStore[T]) GetBatch(ctx context.Context, keys any, result any) ([]any, error) {
	keySlice, err := refl.InterfaceToInterfaceSlice(keys)
	if err != nil {
		return nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	s.recordAttemptedReads(ctx, len(keySlice))
	missing, err := s.KvStore.GetBatch(ctx, keySlice, result)

	if err == nil {
		s.recordReads(ctx, len(keySlice), len(keySlice)-len(missing))
	}

	return missing, err
}

func (s *MetricStore[T]) Put(ctx context.Context, key any, value T) error {
	err := s.KvStore.Put(ctx, key, value)

	if err == nil {
		s.recordWrites(ctx, 1)
	}

	return nil
}

func (s *MetricStore[T]) PutBatch(ctx context.Context, values any) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)
	if err != nil {
		return fmt.Errorf("could not convert values to map[any]any: %w", err)
	}

	err = s.KvStore.PutBatch(ctx, mii)

	if err == nil {
		s.recordWrites(ctx, len(mii))
	}

	return nil
}

func (s *MetricStore[T]) Delete(ctx context.Context, key any) error {
	err := s.KvStore.Delete(ctx, key)

	if err == nil {
		s.recordDeletes(ctx, 1)
	}

	return err
}

func (s *MetricStore[T]) DeleteBatch(ctx context.Context, keys any) error {
	si, err := refl.InterfaceToInterfaceSlice(keys)
	if err != nil {
		return fmt.Errorf("could not convert keys from %T to []any: %w", keys, err)
	}

	err = s.KvStore.DeleteBatch(ctx, si)

	if err == nil {
		s.recordDeletes(ctx, len(si))
	}

	return err
}

func (s *MetricStore[T]) recordSize(sizedStore SizedStore[T]) {
	ticker := time.NewTicker(time.Minute)

	for range ticker.C {
		size := sizedStore.EstimateSize()

		if size != nil {
			s.recordSizeValue(context.Background(), *size)
		}
	}
}

func (s *MetricStore[T]) recordSizeValue(ctx context.Context, size int64) {
	s.metricWriter.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameKvStoreItemCount,
		Dimensions: s.dimensions(),
		Value:      float64(size),
		Unit:       metric.UnitCount,
		Kind:       metric.KindGauge.Build(),
	})
}

func (s *MetricStore[T]) recordAttemptedReads(ctx context.Context, reads int) {
	s.metricWriter.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameKvStoreAttemptedReads,
		Dimensions: s.dimensions(),
		Value:      float64(reads),
		Unit:       metric.UnitCount,
		Kind:       metric.KindCounter.Build(),
	})
}

// recordReads counts successful reads. A read the store served from its own data is the same metric
// told apart by its hit attribute: hits are `reads{hit="true"}`, misses are `reads{hit="false"}`,
// and the total successful read volume is the sum over both.
func (s *MetricStore[T]) recordReads(ctx context.Context, reads int, hits int) {
	s.recordHit(ctx, true, int64(hits))
	s.recordHit(ctx, false, int64(reads-hits))
}

func (s *MetricStore[T]) recordHit(ctx context.Context, hit bool, value int64) {
	dimensions := s.dimensions()
	dimensions[dimensionHit] = strconv.FormatBool(hit)

	s.metricWriter.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameKvStoreReads,
		Dimensions: dimensions,
		Value:      float64(value),
		Unit:       metric.UnitCount,
		Kind:       metric.KindCounter.Build(),
	})
}

func (s *MetricStore[T]) recordWrites(ctx context.Context, count int) {
	s.record(ctx, metricNameKvStoreWrites, int64(count))
}

func (s *MetricStore[T]) recordDeletes(ctx context.Context, count int) {
	s.record(ctx, metricNameKvStoreDeletes, int64(count))
}

func (s *MetricStore[T]) record(ctx context.Context, name string, value int64) {
	s.metricWriter.WriteOne(ctx, &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: name,
		Dimensions: s.dimensions(),
		Value:      float64(value),
	})
}

func (s *MetricStore[T]) dimensions() metric.Dimensions {
	return metric.Dimensions{
		dimensionModelId:   s.model,
		dimensionStoreType: s.store,
	}
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}

// readDefault pre-registers one `reads` series so a dashboard has a line before the first read.
func readDefault(model string, store string, hit bool) *metric.Datum {
	return &metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: metricNameKvStoreReads,
		Dimensions: metric.Dimensions{
			dimensionModelId:   model,
			dimensionStoreType: store,
			dimensionHit:       strconv.FormatBool(hit),
		},
		Unit:  metric.UnitCount,
		Value: 0.0,
		Kind:  metric.KindCounter.Build(),
	}
}

func getDefaultMetrics(model string, store string) metric.Data {
	// no default for the item count, if we don't know the size, it is not 0
	names := []string{
		metricNameKvStoreWrites,
		metricNameKvStoreDeletes,
	}

	defaults := metric.Data{
		readDefault(model, store, true),
		readDefault(model, store, false),
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameKvStoreAttemptedReads,
			Dimensions: metric.Dimensions{
				dimensionModelId:   model,
				dimensionStoreType: store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
			Kind:  metric.KindCounter.Build(),
		},
	}
	for _, name := range names {
		defaults = append(defaults, &metric.Datum{
			Priority:   metric.PriorityHigh,
			MetricName: name,
			Dimensions: map[string]string{
				dimensionModelId:   model,
				dimensionStoreType: store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
			Kind:  metric.KindCounter.Build(),
		})
	}

	return defaults
}
