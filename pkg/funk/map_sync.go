package funk

import (
	"iter"
	"reflect"
	"sync"
)

// NewMapSynced creates an empty concurrency-safe Maper.
func NewMapSynced[K comparable, V any]() Maper[K, V] {
	return &mapSynced[K, V]{
		values: make(map[K]V),
	}
}

var _ Maper[string, int] = &mapSynced[string, int]{}

type mapSynced[K comparable, V any] struct {
	lck    sync.RWMutex
	values map[K]V
}

func (m *mapSynced[K, V]) Size() int {
	m.lck.RLock()
	defer m.lck.RUnlock()

	return len(m.values)
}

func (m *mapSynced[K, V]) IsEmpty() bool {
	m.lck.RLock()
	defer m.lck.RUnlock()

	return len(m.values) == 0
}

func (m *mapSynced[K, V]) ContainsKey(key K) bool {
	m.lck.RLock()
	defer m.lck.RUnlock()

	_, ok := m.values[key]

	return ok
}

func (m *mapSynced[K, V]) ContainsValue(value V) bool {
	m.lck.RLock()
	defer m.lck.RUnlock()

	for _, candidate := range m.values {
		if reflect.DeepEqual(candidate, value) {
			return true
		}
	}

	return false
}

func (m *mapSynced[K, V]) Get(key K) (value V, ok bool) {
	m.lck.RLock()
	defer m.lck.RUnlock()

	value, ok = m.values[key]

	return value, ok
}

func (m *mapSynced[K, V]) Put(key K, value V) (previous V, replaced bool) {
	m.lck.Lock()
	defer m.lck.Unlock()

	previous, replaced = m.values[key]
	m.values[key] = value

	return previous, replaced
}

func (m *mapSynced[K, V]) PutAll(other Maper[K, V]) {
	keys := other.KeySet()
	values := make(map[K]V, len(keys))
	for key := range keys {
		value, _ := other.Get(key)
		values[key] = value
	}

	m.lck.Lock()
	defer m.lck.Unlock()

	for key, value := range values {
		m.values[key] = value
	}
}

func (m *mapSynced[K, V]) Merge(other ...Maper[K, V]) Maper[K, V] {
	merged := NewMapSynced[K, V]()
	merged.PutAll(m)
	for _, item := range other {
		merged.PutAll(item)
	}

	return merged
}

func (m *mapSynced[K, V]) MergeWith(combine func(V, V) V, other ...Maper[K, V]) Maper[K, V] {
	merged := NewMapSynced[K, V]()
	merged.PutAll(m)
	for _, item := range other {
		for key := range item.KeySet() {
			value, _ := item.Get(key)
			if existing, ok := merged.Get(key); ok {
				merged.Put(key, combine(existing, value))

				continue
			}

			merged.Put(key, value)
		}
	}

	return merged
}

func (m *mapSynced[K, V]) Intersect(other ...Maper[K, V]) Maper[K, V] {
	intersection := NewMapSynced[K, V]()
	for key := range m.KeySet() {
		if All(other, func(item Maper[K, V]) bool {
			return item.ContainsKey(key)
		}) {
			value, _ := m.Get(key)
			intersection.Put(key, value)
		}
	}

	return intersection
}

func (m *mapSynced[K, V]) IntersectWith(combine func(V, V) V, other ...Maper[K, V]) Maper[K, V] {
	intersection := m.Intersect(other...)
	for key := range intersection.KeySet() {
		value, _ := intersection.Get(key)
		for _, item := range other {
			otherValue, _ := item.Get(key)
			value = combine(value, otherValue)
		}
		intersection.Put(key, value)
	}

	return intersection
}

func (m *mapSynced[K, V]) Difference(other Maper[K, V]) (inThis Maper[K, V], inOther Maper[K, V]) {
	inThis, inOther = NewMapSynced[K, V](), NewMapSynced[K, V]()
	for key := range m.KeySet() {
		if !other.ContainsKey(key) {
			value, _ := m.Get(key)
			inThis.Put(key, value)
		}
	}
	for key := range other.KeySet() {
		if !m.ContainsKey(key) {
			value, _ := other.Get(key)
			inOther.Put(key, value)
		}
	}

	return inThis, inOther
}

func (m *mapSynced[K, V]) Remove(key K) (previous V, removed bool) {
	m.lck.Lock()
	defer m.lck.Unlock()

	previous, removed = m.values[key]
	delete(m.values, key)

	return previous, removed
}

func (m *mapSynced[K, V]) Clear() {
	m.lck.Lock()
	defer m.lck.Unlock()

	clear(m.values)
}

func (m *mapSynced[K, V]) KeySet() Set[K] {
	m.lck.RLock()
	defer m.lck.RUnlock()

	keys := make(Set[K], len(m.values))
	for key := range m.values {
		keys[key] = struct{}{}
	}

	return keys
}

func (m *mapSynced[K, V]) Keys() []K {
	return m.KeySet().ToSlice()
}

func (m *mapSynced[K, V]) Values() []V {
	m.lck.RLock()
	defer m.lck.RUnlock()

	values := make([]V, 0, len(m.values))
	for _, value := range m.values {
		values = append(values, value)
	}

	return values
}

func (m *mapSynced[K, V]) Range() iter.Seq2[K, V] {
	m.lck.RLock()
	defer m.lck.RUnlock()

	entries := make([]mapEntry[K, V], 0, len(m.values))
	for key, value := range m.values {
		entries = append(entries, mapEntry[K, V]{key: key, value: value})
	}

	return func(yield func(K, V) bool) {
		for _, entry := range entries {
			if !yield(entry.key, entry.value) {
				return
			}
		}
	}
}

func (m *mapSynced[K, V]) Filter(filter func(key K, value V) bool) Maper[K, V] {
	m.lck.RLock()
	defer m.lck.RUnlock()

	filtered := NewMapSynced[K, V]()
	for key, value := range m.values {
		if filter(key, value) {
			filtered.Put(key, value)
		}
	}

	return filtered
}

func (m *mapSynced[K, V]) Any(predicate func(key K, value V) bool) bool {
	m.lck.RLock()
	defer m.lck.RUnlock()

	for key, value := range m.values {
		if predicate(key, value) {
			return true
		}
	}

	return false
}
