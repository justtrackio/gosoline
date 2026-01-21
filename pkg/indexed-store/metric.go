package indexed_store

import (
	"context"
	"fmt"
	"github.com/applike/gosoline/pkg/mdl"
	"github.com/applike/gosoline/pkg/metric"
	"github.com/applike/gosoline/pkg/refl"
	"time"
)

const (
	// number of items stored in the store (if available)
	metricNameIndexedStoreSize = "indexedStoreSize"
	// number of items we try to read from the store
	metricNameIndexedStoreRead = "indexedStoreRead"
	// number of items found and read from the store
	metricNameIndexedStoreHit = "indexedStoreHit"
	// number of items written to the store
	metricNameIndexedStoreWrite = "indexedStoreWrite"
	// number of items deleted from the store
	metricNameIndexedStoreDelete = "indexedStoreDelete"
)

type MetricStore struct {
	baseStore    IndexedStore
	metricWriter metric.Writer
	model        string
	store        string
}

func NewMetricStoreWithInterfaces(store IndexedStore, settings *Settings) IndexedStore {
	if !settings.MetricsEnabled {
		return store
	}

	storeName := fmt.Sprintf("%T", store)
	model := (&mdl.ModelId{
		Project:     settings.Project,
		Environment: settings.Environment,
		Family:      settings.Family,
		Application: settings.Application,
		Name:        settings.Name,
	}).String()
	defaults := getDefaultMetrics(model, storeName)

	s := &MetricStore{
		baseStore:    store,
		metricWriter: metric.NewDaemonWriter(defaults...),
		model:        model,
		store:        storeName,
	}

	if sizedStore, ok := store.(SizedStore); ok {
		go s.recordSize(sizedStore)
	}

	return s
}

func (s *MetricStore) Contains(ctx context.Context, key interface{}) (bool, error) {
	s.recordReads(1)

	found, err := s.baseStore.Contains(ctx, key)

	if found && err == nil {
		s.recordHits(1)
	}

	return found, err
}

func (s *MetricStore) ContainsInIndex(ctx context.Context, index string, key interface{}, rangeKeys ...interface{}) (bool, error) {
	s.recordIndexReads(index, 1)

	found, err := s.baseStore.ContainsInIndex(ctx, index, key, rangeKeys...)

	if found && err == nil {
		s.recordIndexHits(index, 1)
	}

	return found, err
}

func (s *MetricStore) Get(ctx context.Context, key interface{}) (BaseValue, error) {
	s.recordReads(1)

	value, err := s.baseStore.Get(ctx, key)

	if value != nil && err == nil {
		s.recordHits(1)
	}

	return value, err
}

func (s *MetricStore) GetFromIndex(ctx context.Context, index string, key interface{}, rangeKeys ...interface{}) (BaseValue, error) {
	s.recordIndexReads(index, 1)

	value, err := s.baseStore.GetFromIndex(ctx, index, key, rangeKeys...)

	if value != nil && err == nil {
		s.recordIndexHits(index, 1)
	}

	return value, err
}

func (s *MetricStore) GetBatch(ctx context.Context, keys interface{}) ([]BaseValue, error) {
	keySlice, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	s.recordReads(len(keySlice))

	result, err := s.baseStore.GetBatch(ctx, keySlice)

	if err == nil {
		s.recordHits(len(result))
	}

	return result, err
}

func (s *MetricStore) GetBatchFromIndex(ctx context.Context, index string, keys interface{}, rangeKeys ...interface{}) ([]BaseValue, error) {
	keySlice, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	s.recordIndexReads(index, len(keySlice))

	result, err := s.baseStore.GetBatchFromIndex(ctx, index, keySlice, rangeKeys...)

	if err == nil {
		s.recordIndexHits(index, len(result))
	}

	return result, err
}

func (s *MetricStore) GetBatchWithMissing(ctx context.Context, keys interface{}) ([]BaseValue, []interface{}, error) {
	keySlice, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return nil, nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	s.recordReads(len(keySlice))

	result, missing, err := s.baseStore.GetBatchWithMissing(ctx, keySlice)

	if err == nil {
		s.recordHits(len(result))
	}

	return result, missing, err
}

func (s *MetricStore) GetBatchWithMissingFromIndex(ctx context.Context, index string, keys interface{}, rangeKeys ...interface{}) ([]BaseValue, []MissingValue, error) {
	keySlice, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return nil, nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	s.recordIndexReads(index, len(keySlice))

	result, missing, err := s.baseStore.GetBatchWithMissingFromIndex(ctx, index, keySlice, rangeKeys...)

	if err == nil {
		s.recordIndexHits(index, len(result))
	}

	return result, missing, err
}

func (s *MetricStore) Put(ctx context.Context, value BaseValue) error {
	err := s.baseStore.Put(ctx, value)

	if err == nil {
		s.recordWrites(1)
	}

	return nil
}

func (s *MetricStore) PutBatch(ctx context.Context, values interface{}) error {
	mii, err := refl.InterfaceToMapInterfaceInterface(values)

	if err != nil {
		return fmt.Errorf("could not convert values to map[interface{}]interface{}: %w", err)
	}

	err = s.baseStore.PutBatch(ctx, mii)

	if err == nil {
		s.recordWrites(len(mii))
	}

	return nil
}

func (s *MetricStore) Delete(ctx context.Context, key interface{}) error {
	err := s.baseStore.Delete(ctx, key)

	if err == nil {
		s.recordDeletes(1)
	}

	return err
}

func (s *MetricStore) DeleteBatch(ctx context.Context, keys interface{}) error {
	si, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return fmt.Errorf("could not convert keys from %T to []interface{}: %w", keys, err)
	}

	err = s.baseStore.DeleteBatch(ctx, si)

	if err == nil {
		s.recordDeletes(len(si))
	}

	return err
}

func (s *MetricStore) recordSize(sizedStore SizedStore) {
	ticker := time.NewTicker(time.Minute)

	for range ticker.C {
		size := sizedStore.EstimateSize()

		if size != nil {
			s.record(metricNameIndexedStoreSize, nil, int(*size))
		}
	}
}

func (s *MetricStore) recordReads(count int) {
	s.record(metricNameIndexedStoreRead, nil, count)
}

func (s *MetricStore) recordIndexReads(index string, count int) {
	s.record(metricNameIndexedStoreRead, &index, count)
}

func (s *MetricStore) recordHits(count int) {
	s.record(metricNameIndexedStoreHit, nil, count)
}

func (s *MetricStore) recordIndexHits(index string, count int) {
	s.record(metricNameIndexedStoreHit, &index, count)
}

func (s *MetricStore) recordWrites(count int) {
	s.record(metricNameIndexedStoreWrite, nil, count)
}

func (s *MetricStore) recordDeletes(count int) {
	s.record(metricNameIndexedStoreDelete, nil, count)
}

func (s *MetricStore) record(name string, index *string, value int) {
	dimensions := map[string]string{
		"model": s.model,
		"store": s.store,
	}
	if index != nil {
		dimensions["index"] = *index
	}

	s.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: name,
		Dimensions: dimensions,
		Value:      float64(value),
		Unit:       metric.UnitCount,
	})
}

func getDefaultMetrics(model string, store string) metric.Data {
	// no default for the size, if we don't know the size, it is not 0

	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameIndexedStoreRead,
			Dimensions: map[string]string{
				"model": model,
				"store": store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameIndexedStoreHit,
			Dimensions: map[string]string{
				"model": model,
				"store": store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameIndexedStoreWrite,
			Dimensions: map[string]string{
				"model": model,
				"store": store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameIndexedStoreDelete,
			Dimensions: map[string]string{
				"model": model,
				"store": store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
