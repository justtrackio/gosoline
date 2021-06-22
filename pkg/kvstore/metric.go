package kvstore

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
	metricNameKvStoreSize = "kvStoreSize"
	// number of items we try to read from the store
	metricNameKvStoreRead = "kvStoreRead"
	// number of items found and read from the store
	metricNameKvStoreHit = "kvStoreHit"
	// number of items written to the store
	metricNameKvStoreWrite = "kvStoreWrite"
	// number of items deleted from the store
	metricNameKvStoreDelete = "kvStoreDelete"
)

type MetricStore struct {
	KvStore
	metricWriter metric.Writer
	model        string
	store        string
}

func NewMetricStoreWithInterfaces(store KvStore, settings *Settings) KvStore {
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
		KvStore:      store,
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

	found, err := s.KvStore.Contains(ctx, key)

	if found && err == nil {
		s.recordHits(1)
	}

	return found, err
}

func (s *MetricStore) Get(ctx context.Context, key interface{}, value interface{}) (bool, error) {
	s.recordReads(1)

	found, err := s.KvStore.Get(ctx, key, value)

	if found && err == nil {
		s.recordHits(1)
	}

	return found, err
}

func (s *MetricStore) GetBatch(ctx context.Context, keys interface{}, result interface{}) ([]interface{}, error) {
	keySlice, err := refl.InterfaceToInterfaceSlice(keys)

	if err != nil {
		return nil, fmt.Errorf("can not morph keys to slice of interfaces: %w", err)
	}

	s.recordReads(len(keySlice))

	missing, err := s.KvStore.GetBatch(ctx, keySlice, result)

	if err == nil {
		s.recordHits(len(keySlice) - len(missing))
	}

	return missing, err
}

func (s *MetricStore) Put(ctx context.Context, key interface{}, value interface{}) error {
	err := s.KvStore.Put(ctx, key, value)

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

	err = s.KvStore.PutBatch(ctx, mii)

	if err == nil {
		s.recordWrites(len(mii))
	}

	return nil
}

func (s *MetricStore) Delete(ctx context.Context, key interface{}) error {
	err := s.KvStore.Delete(ctx, key)

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

	err = s.KvStore.DeleteBatch(ctx, si)

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
			s.record(metricNameKvStoreSize, *size)
		}
	}
}

func (s *MetricStore) recordReads(count int) {
	s.record(metricNameKvStoreRead, int64(count))
}

func (s *MetricStore) recordHits(count int) {
	s.record(metricNameKvStoreHit, int64(count))
}

func (s *MetricStore) recordWrites(count int) {
	s.record(metricNameKvStoreWrite, int64(count))
}

func (s *MetricStore) recordDeletes(count int) {
	s.record(metricNameKvStoreDelete, int64(count))
}

func (s *MetricStore) record(name string, value int64) {
	s.metricWriter.WriteOne(&metric.Datum{
		Priority:   metric.PriorityHigh,
		MetricName: name,
		Dimensions: map[string]string{
			"model": s.model,
			"store": s.store,
		},
		Value: float64(value),
		Unit:  metric.UnitCount,
	})
}

func getDefaultMetrics(model string, store string) metric.Data {
	// no default for the size, if we don't know the size, it is not 0

	return metric.Data{
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameKvStoreRead,
			Dimensions: map[string]string{
				"model": model,
				"store": store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameKvStoreHit,
			Dimensions: map[string]string{
				"model": model,
				"store": store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameKvStoreWrite,
			Dimensions: map[string]string{
				"model": model,
				"store": store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
		{
			Priority:   metric.PriorityHigh,
			MetricName: metricNameKvStoreDelete,
			Dimensions: map[string]string{
				"model": model,
				"store": store,
			},
			Unit:  metric.UnitCount,
			Value: 0.0,
		},
	}
}
