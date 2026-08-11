package funk_test

import (
	"sync"
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/suite"
)

type MapSyncedTestSuite struct {
	suite.Suite
	mapper funk.Maper[string, int]
}

func TestMapSyncedTestSuite(t *testing.T) {
	suite.Run(t, new(MapSyncedTestSuite))
}

func (s *MapSyncedTestSuite) SetupTest() {
	s.mapper = funk.NewMapSynced[string, int]()
}

func (s *MapSyncedTestSuite) TestMapOperations() {
	previous, replaced := s.mapper.Put("first", 1)
	s.Zero(previous)
	s.False(replaced)

	previous, replaced = s.mapper.Put("first", 2)
	s.Equal(1, previous)
	s.True(replaced)
	s.True(s.mapper.ContainsKey("first"))
	s.True(s.mapper.ContainsValue(2))
	s.Equal(1, s.mapper.Size())

	value, ok := s.mapper.Get("first")
	s.True(ok)
	s.Equal(2, value)

	previous, removed := s.mapper.Remove("first")
	s.True(removed)
	s.Equal(2, previous)
	s.True(s.mapper.IsEmpty())
}

func (s *MapSyncedTestSuite) TestPutAllAndSnapshots() {
	s.mapper.Put("first", 1)
	s.mapper.PutAll(s.mapper)

	other := funk.NewMapSynced[string, int]()
	other.Put("first", 2)
	other.Put("second", 3)
	s.mapper.PutAll(other)

	keys := s.mapper.KeySet()
	values := s.mapper.Values()
	s.ElementsMatch([]string{"first", "second"}, keys.ToSlice())
	s.ElementsMatch([]int{2, 3}, values)

	keys.Remove("first")
	values[0] = 99
	s.True(s.mapper.ContainsKey("first"))
	s.False(s.mapper.ContainsValue(99))

	s.mapper.Clear()
	s.True(s.mapper.IsEmpty())
}

func (s *MapSyncedTestSuite) TestMergeIntersectAndDifference() {
	s.mapper.Put("first", 1)
	s.mapper.Put("shared", 2)
	other := funk.NewMapSynced[string, int]()
	other.Put("shared", 3)
	other.Put("second", 4)

	merged := s.mapper.Merge(other)
	combined := s.mapper.MergeWith(func(left, right int) int { return left + right }, other)
	intersection := s.mapper.Intersect(other)
	intersected := s.mapper.IntersectWith(func(left, right int) int { return left + right }, other)
	inThis, inOther := s.mapper.Difference(other)

	s.Equal(2, s.mapper.Size())
	s.Equal(3, merged.Size())
	s.Equal(3, s.mapValue(merged, "shared"))
	s.Equal(5, s.mapValue(combined, "shared"))
	s.Equal(2, s.mapValue(intersection, "shared"))
	s.Equal(5, s.mapValue(intersected, "shared"))
	s.Equal(1, s.mapValue(inThis, "first"))
	s.Equal(4, s.mapValue(inOther, "second"))
}

func (s *MapSyncedTestSuite) TestFilter() {
	s.mapper.Put("first", 1)
	s.mapper.Put("second", 2)
	s.mapper.Put("third", 3)

	filtered := s.mapper.Filter(func(_ string, value int) bool {
		return value%2 == 0
	})

	s.NotSame(s.mapper, filtered)
	s.Equal(3, s.mapper.Size())
	s.Equal(1, filtered.Size())
	s.True(s.mapper.ContainsKey("first"))
	s.False(filtered.ContainsKey("first"))
	s.True(filtered.ContainsKey("second"))
	s.False(filtered.ContainsKey("third"))
}

func (s *MapSyncedTestSuite) TestConcurrentAccess() {
	const workers = 100

	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			key := string(rune(i))
			s.mapper.Put(key, i)
			s.mapper.Get(key)
			s.mapper.ContainsKey(key)
			s.mapper.ContainsValue(i)
			s.mapper.KeySet()
			s.mapper.Values()
			s.mapper.Size()
			s.mapper.IsEmpty()
		}()
	}
	wg.Wait()

	s.Equal(workers, s.mapper.Size())
}

func (s *MapSyncedTestSuite) TestKeys() {
	s.mapper.Put("first", 1)
	s.mapper.Put("second", 2)

	s.ElementsMatch([]string{"first", "second"}, s.mapper.Keys())
}

func (s *MapSyncedTestSuite) TestRange() {
	s.mapper.Put("first", 1)
	s.mapper.Put("second", 2)

	entries := s.mapper.Range()
	s.mapper.Clear()

	actual := make(map[string]int)
	for key, value := range entries {
		actual[key] = value
	}

	s.Equal(map[string]int{"first": 1, "second": 2}, actual)
}

func (s *MapSyncedTestSuite) mapValue(mapper funk.Maper[string, int], key string) int {
	value, ok := mapper.Get(key)
	s.True(ok)

	return value
}
