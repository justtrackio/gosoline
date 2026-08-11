package funk_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/suite"
)

type MapperTestSuite struct {
	suite.Suite
	mapper funk.Maper[string, int]
}

func TestMapperTestSuite(t *testing.T) {
	suite.Run(t, new(MapperTestSuite))
}

func (s *MapperTestSuite) SetupTest() {
	s.mapper = funk.NewMaper[string, int]()
}

func (s *MapperTestSuite) TestNewMapper() {
	s.True(s.mapper.IsEmpty())
	s.Equal(0, s.mapper.Size())
	s.Empty(s.mapper.KeySet())
	s.Empty(s.mapper.Values())
}

func (s *MapperTestSuite) TestPutAndGet() {
	previous, replaced := s.mapper.Put("zero", 0)
	s.False(replaced)
	s.Zero(previous)
	s.True(s.mapper.ContainsKey("zero"))
	s.True(s.mapper.ContainsValue(0))
	s.False(s.mapper.ContainsKey("missing"))
	s.False(s.mapper.ContainsValue(1))
	s.Equal(1, s.mapper.Size())
	s.False(s.mapper.IsEmpty())

	value, ok := s.mapper.Get("zero")
	s.True(ok)
	s.Zero(value)

	value, ok = s.mapper.Get("missing")
	s.False(ok)
	s.Zero(value)

	previous, replaced = s.mapper.Put("zero", 1)
	s.True(replaced)
	s.Zero(previous)
	s.Equal(1, s.mapper.Size())
	s.True(s.mapper.ContainsValue(1))
}

func (s *MapperTestSuite) TestPutAll() {
	s.mapper.Put("existing", 1)
	other := funk.NewMaper[string, int]()
	other.Put("existing", 2)
	other.Put("new", 3)

	s.mapper.PutAll(other)

	s.Equal(2, s.mapper.Size())
	s.Equal(2, s.value("existing"))
	s.Equal(3, s.value("new"))
}

func (s *MapperTestSuite) TestMerge() {
	s.mapper.Put("first", 1)
	other := funk.NewMaper[string, int]()
	other.Put("first", 2)
	other.Put("second", 3)

	merged := s.mapper.Merge(other)
	combined := s.mapper.MergeWith(func(left, right int) int { return left + right }, other)

	s.Equal(1, s.mapper.Size())
	s.Equal(2, merged.Size())
	s.Equal(2, s.mapValue(merged, "first"))
	s.Equal(3, s.mapValue(merged, "second"))
	s.Equal(3, s.mapValue(combined, "first"))
	s.Equal(3, s.mapValue(combined, "second"))
}

func (s *MapperTestSuite) TestIntersectAndDifference() {
	s.mapper.Put("first", 1)
	s.mapper.Put("shared", 2)
	other := funk.NewMaper[string, int]()
	other.Put("shared", 3)
	other.Put("second", 4)

	intersection := s.mapper.Intersect(other)
	combined := s.mapper.IntersectWith(func(left, right int) int { return left + right }, other)
	inThis, inOther := s.mapper.Difference(other)

	s.Equal(1, intersection.Size())
	s.Equal(2, s.mapValue(intersection, "shared"))
	s.Equal(5, s.mapValue(combined, "shared"))
	s.Equal(1, s.mapValue(inThis, "first"))
	s.Equal(4, s.mapValue(inOther, "second"))
}

func (s *MapperTestSuite) TestRemoveAndClear() {
	s.mapper.Put("first", 1)
	s.mapper.Put("second", 2)

	previous, removed := s.mapper.Remove("first")
	s.True(removed)
	s.Equal(1, previous)
	s.False(s.mapper.ContainsKey("first"))

	previous, removed = s.mapper.Remove("missing")
	s.False(removed)
	s.Zero(previous)

	s.mapper.Clear()
	s.True(s.mapper.IsEmpty())
	s.Equal(0, s.mapper.Size())
}

func (s *MapperTestSuite) TestKeySetAndValuesAreSnapshots() {
	s.mapper.Put("first", 1)
	s.mapper.Put("second", 2)

	keys := s.mapper.KeySet()
	keySlice := s.mapper.Keys()
	values := s.mapper.Values()

	s.ElementsMatch([]string{"first", "second"}, keys.ToSlice())
	s.ElementsMatch([]string{"first", "second"}, keySlice)
	s.ElementsMatch([]int{1, 2}, values)

	keys.Remove("first")
	values[0] = 99

	s.True(s.mapper.ContainsKey("first"))
	s.False(s.mapper.ContainsValue(99))
}

func (s *MapperTestSuite) TestFilter() {
	s.mapper.Put("first", 1)
	s.mapper.Put("second", 2)
	s.mapper.Put("third", 3)

	filtered := s.mapper.Filter(func(_ string, value int) bool {
		return value%2 == 1
	})

	s.NotSame(s.mapper, filtered)
	s.Equal(3, s.mapper.Size())
	s.Equal(2, filtered.Size())
	s.True(s.mapper.ContainsKey("second"))
	s.True(filtered.ContainsKey("first"))
	s.False(filtered.ContainsKey("second"))
	s.True(filtered.ContainsKey("third"))
}

func (s *MapperTestSuite) TestRange() {
	s.mapper.Put("first", 1)
	s.mapper.Put("second", 2)

	entries := s.mapper.Range()
	s.mapper.Put("third", 3)

	actual := make(map[string]int)
	for key, value := range entries {
		actual[key] = value
	}

	s.Equal(map[string]int{"first": 1, "second": 2}, actual)
}

func (s *MapperTestSuite) value(key string) int {
	return s.mapValue(s.mapper, key)
}

func (s *MapperTestSuite) mapValue(mapper funk.Maper[string, int], key string) int {
	value, ok := mapper.Get(key)
	s.True(ok)

	return value
}
