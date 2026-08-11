package funk

import (
	"iter"
	"reflect"
)

// Maper describes a mutable collection of key-value mappings.
//
// Methods that return a stored value also return whether a mapping was present,
// so a stored zero value is distinguishable from an absent key. KeySet and
// Values return snapshots; modifying them does not modify the Maper.
type Maper[K comparable, V any] interface {
	// Size returns the number of key-value mappings.
	Size() int
	// IsEmpty reports whether the Maper contains no mappings.
	IsEmpty() bool
	// ContainsKey reports whether key has a mapping.
	ContainsKey(key K) bool
	// ContainsValue reports whether value is mapped by at least one key.
	ContainsValue(value V) bool
	// Get returns the value mapped by key and whether the mapping exists.
	Get(key K) (value V, ok bool)
	// Put associates value with key and returns the previous value and whether it was replaced.
	Put(key K, value V) (previous V, replaced bool)
	// PutAll copies all mappings from other into the Maper.
	PutAll(other Maper[K, V])
	// Merge returns a new Maper containing the mappings from this Maper followed by other.
	Merge(other ...Maper[K, V]) Maper[K, V]
	// MergeWith returns a new Maper, combining values for duplicate keys.
	MergeWith(combine func(V, V) V, other ...Maper[K, V]) Maper[K, V]
	// Intersect returns a new Maper containing this Maper's mappings whose keys are present in every other Maper.
	Intersect(other ...Maper[K, V]) Maper[K, V]
	// IntersectWith returns a new Maper containing values combined for keys present in every Maper.
	IntersectWith(combine func(V, V) V, other ...Maper[K, V]) Maper[K, V]
	// Difference returns mappings unique to this Maper and mappings unique to other.
	Difference(other Maper[K, V]) (inThis Maper[K, V], inOther Maper[K, V])
	// Remove removes key's mapping and returns its previous value and whether it existed.
	Remove(key K) (previous V, removed bool)
	// Clear removes all mappings.
	Clear()
	// KeySet returns a snapshot of the mapped keys.
	KeySet() Set[K]
	// Keys returns a snapshot of the mapped keys in undefined order.
	Keys() []K
	// Values returns a snapshot of the mapped values.
	Values() []V
	// Range returns an iterator over a snapshot of the mappings in undefined order.
	Range() iter.Seq2[K, V]
	// Any reports whether at least one mapping satisfies the predicate. The order of predicate calls is undefined.
	Any(predicate func(key K, value V) bool) bool
	// Filter creates and returns a new map with entries removed which don't satisfy a predicate. The order of the calls to the predicate is undefined.
	Filter(filter func(key K, value V) bool) Maper[K, V]
}

// NewMaper creates an empty Maper.
func NewMaper[K comparable, V any]() Maper[K, V] {
	return &maper[K, V]{
		values: make(map[K]V),
	}
}

var _ Maper[string, int] = &maper[string, int]{}

type maper[K comparable, V any] struct {
	values map[K]V
}

type mapEntry[K comparable, V any] struct {
	key   K
	value V
}

func (m *maper[K, V]) Size() int {
	return len(m.values)
}

func (m *maper[K, V]) IsEmpty() bool {
	return len(m.values) == 0
}

func (m *maper[K, V]) ContainsKey(key K) bool {
	_, ok := m.values[key]

	return ok
}

func (m *maper[K, V]) ContainsValue(value V) bool {
	for _, candidate := range m.values {
		if reflect.DeepEqual(candidate, value) {
			return true
		}
	}

	return false
}

func (m *maper[K, V]) Get(key K) (value V, ok bool) {
	value, ok = m.values[key]

	return value, ok
}

func (m *maper[K, V]) Put(key K, value V) (previous V, replaced bool) {
	previous, replaced = m.values[key]
	m.values[key] = value

	return previous, replaced
}

func (m *maper[K, V]) PutAll(other Maper[K, V]) {
	for key := range other.KeySet() {
		value, _ := other.Get(key)
		m.values[key] = value
	}
}

func (m *maper[K, V]) Merge(other ...Maper[K, V]) Maper[K, V] {
	merged := NewMaper[K, V]()
	merged.PutAll(m)
	for _, item := range other {
		merged.PutAll(item)
	}

	return merged
}

func (m *maper[K, V]) MergeWith(combine func(V, V) V, other ...Maper[K, V]) Maper[K, V] {
	merged := NewMaper[K, V]()
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

func (m *maper[K, V]) Intersect(other ...Maper[K, V]) Maper[K, V] {
	intersection := NewMaper[K, V]()
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

func (m *maper[K, V]) IntersectWith(combine func(V, V) V, other ...Maper[K, V]) Maper[K, V] {
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

func (m *maper[K, V]) Difference(other Maper[K, V]) (inThis Maper[K, V], inOther Maper[K, V]) {
	inThis, inOther = NewMaper[K, V](), NewMaper[K, V]()
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

func (m *maper[K, V]) Remove(key K) (previous V, removed bool) {
	previous, removed = m.values[key]
	delete(m.values, key)

	return previous, removed
}

func (m *maper[K, V]) Clear() {
	clear(m.values)
}

func (m *maper[K, V]) KeySet() Set[K] {
	keys := make(Set[K], len(m.values))
	for key := range m.values {
		keys[key] = struct{}{}
	}

	return keys
}

func (m *maper[K, V]) Keys() []K {
	return m.KeySet().ToSlice()
}

func (m *maper[K, V]) Values() []V {
	values := make([]V, 0, len(m.values))
	for _, value := range m.values {
		values = append(values, value)
	}

	return values
}

func (m *maper[K, V]) Range() iter.Seq2[K, V] {
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

func (m *maper[K, V]) Filter(filter func(key K, value V) bool) Maper[K, V] {
	filtered := NewMaper[K, V]()
	for key, value := range m.values {
		if filter(key, value) {
			filtered.Put(key, value)
		}
	}

	return filtered
}

func (m *maper[K, V]) Any(predicate func(key K, value V) bool) bool {
	for key, value := range m.values {
		if predicate(key, value) {
			return true
		}
	}

	return false
}
