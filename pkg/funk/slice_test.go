package funk_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

func TestCastSlice(t *testing.T) {
	inputSlice := []any{"", ""}
	expectedSlice := []string{"", ""}

	target, err := funk.CastSlice[string](inputSlice)

	assert.Equal(t, nil, err)
	assert.Equal(t, expectedSlice, target)
}

func TestContains(t *testing.T) {
	type test struct {
		Foo string
	}

	in := []test{{
		Foo: "bar",
	}}

	out := funk.Contains(in, test{Foo: "bar"})
	assert.True(t, out)
}

func TestContainsAll(t *testing.T) {
	cases := map[string]struct {
		in       []int
		elements []int
		exp      bool
	}{
		"contains all equal": {
			in:       []int{1, 2, 3},
			elements: []int{2, 1, 3},
			exp:      true,
		},
		"contains all not equal": {
			in:       []int{3, 1, 2},
			elements: []int{1, 3},
			exp:      true,
		},
		"does not contain all": {
			in:       []int{1, 2},
			elements: []int{1, 2, 3},
			exp:      false,
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			out := funk.ContainsAll(c.in, c.elements)
			assert.Equal(t, c.exp, out)
		})
	}
}

func TestChunk(t *testing.T) {
	type input struct {
		Sl   []int
		Size int
	}

	tests := map[string]struct {
		Name string
		In   input
		Out  [][]int
	}{
		"remainder one": {
			In: input{
				Sl:   []int{0, 1, 2, 3, 4, 5, 6},
				Size: 3,
			},
			Out: [][]int{
				{0, 1, 2},
				{3, 4, 5},
				{6},
			},
		},
		"remainder two": {
			In: input{
				Sl:   []int{0, 1, 2, 3, 4, 5, 6, 7},
				Size: 3,
			},
			Out: [][]int{
				{0, 1, 2},
				{3, 4, 5},
				{6, 7},
			},
		},
		"size negative": {
			In: input{
				Sl:   []int{0, 1, 2, 3, 4, 5, 6},
				Size: -3,
			},
			Out: nil,
		},
		"size zero": {
			In: input{
				Sl:   []int{0, 1, 2, 3, 4, 5, 6},
				Size: 0,
			},
			Out: nil,
		},
		"size one": {
			In: input{
				Sl:   []int{0, 1, 2, 3},
				Size: 1,
			},
			Out: [][]int{{0}, {1}, {2}, {3}},
		},
		"size larger than slice": {
			In: input{
				Sl:   []int{0, 1, 2, 3},
				Size: 5,
			},
			Out: [][]int{{0, 1, 2, 3}},
		},
		"no values": {
			In: input{
				Sl:   []int{},
				Size: 5,
			},
			Out: [][]int{},
		},
	}
	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			res := funk.Chunk(data.In.Sl, data.In.Size)
			assert.Equalf(t, data.Out, res, "Test static failed: %s", data.Name)
		})
	}
}

func TestDifference(t *testing.T) {
	tests := map[string]struct {
		Input1 []int
		Input2 []int

		Out1 []int
		Out2 []int
	}{
		"simple": {
			Input1: []int{1, 2, 3, 4},
			Input2: []int{2, 3, 5, 6},
			Out1:   []int{1, 4},
			Out2:   []int{5, 6},
		},
		"identical": {
			Input1: []int{1, 2, 3},
			Input2: []int{1, 2, 3},
			Out1:   []int{},
			Out2:   []int{},
		},
		"disjunct": {
			Input1: []int{1, 2},
			Input2: []int{3, 4},
			Out1:   []int{1, 2},
			Out2:   []int{3, 4},
		},
		"left empty": {
			Input1: []int{},
			Input2: []int{3, 4},
			Out1:   []int{},
			Out2:   []int{3, 4},
		},
		"right empty": {
			Input1: []int{1, 2},
			Input2: []int{},
			Out1:   []int{1, 2},
			Out2:   []int{},
		},
	}

	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			l, r := funk.Difference(data.Input1, data.Input2)
			assert.ElementsMatch(t, l, data.Out1)
			assert.ElementsMatch(t, r, data.Out2)
		})
	}
}

func TestFlatten(t *testing.T) {
	tl := [][]string{{"foo", "bar"}, {"raz"}}
	expected := []string{"foo", "bar", "raz"}

	tlf := funk.Flatten(tl)

	assert.Equal(t, expected, tlf)
}

func TestIndex(t *testing.T) {
	type obj struct {
		Foo string
	}

	tests := map[string]struct {
		in    []obj
		index int
	}{
		"exists": {
			in:    []obj{{Foo: "bar"}},
			index: 0,
		},
		"missing": {
			in:    []obj{{Foo: "foo"}},
			index: -1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			out := funk.Index(test.in, obj{Foo: "bar"})

			assert.Equal(t, test.index, out)
		})
	}
}

func TestIntersect(t *testing.T) {
	tests := map[string]struct {
		Input1 []int
		Input2 []int

		Out []int
	}{
		"simple": {
			Input1: []int{1: 2, 3, 4},
			Input2: []int{2, 3, 5, 6},
			Out:    []int{2, 3},
		},
		"identical": {
			Input1: []int{1, 2, 3},
			Input2: []int{1, 2, 3},
			Out:    []int{1, 2, 3},
		},
		"disjunct": {
			Input1: []int{1, 2},
			Input2: []int{3, 4},
			Out:    []int{},
		},
		"left empty": {
			Input1: []int{},
			Input2: []int{3, 4},
			Out:    []int{},
		},
		"right empty": {
			Input1: []int{1, 2},
			Input2: []int{},
			Out:    []int{},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			res := funk.Intersect(test.Input1, test.Input2)
			assert.ElementsMatch(t, res, test.Out)
		})
	}
}

func TestMapEmptyInterface(t *testing.T) {
	type test []interface{}
	tl := test{
		"blah", "test",
	}

	tlf := funk.Map(tl, func(i interface{}) string {
		return i.(string)
	})
	assert.True(t, slices.Contains(tlf, "blah"))
}

func TestRepeatPrimitive(t *testing.T) {
	tests := map[string]struct {
		Times   int
		Element int

		Out []int
	}{
		"simple": {
			Times:   5,
			Element: 1,
			Out:     []int{1, 1, 1, 1, 1},
		},
		"slice len 0": {
			Times:   0,
			Element: 1,
			Out:     []int{},
		},
		"single": {
			Times:   1,
			Element: 1,
			Out:     []int{1},
		},
		"empty value as input": {
			Times:   5,
			Element: 0,
			Out:     []int{0, 0, 0, 0, 0},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			out := funk.Repeat(test.Element, test.Times)
			assert.Equal(t, out, test.Out)
		})
	}
}

func TestRepeatStructPointer(t *testing.T) {
	type test struct{ field int }

	test1 := &test{1}

	tests := map[string]struct {
		Times   int
		Element *test

		Out []*test
	}{
		"simple": {
			Times:   5,
			Element: test1,
			Out:     []*test{test1, test1, test1, test1, test1},
		},
		"nil": {
			Times:   5,
			Element: nil,
			Out:     []*test{nil, nil, nil, nil, nil},
		},
		"nil wit empty slice": {
			Times:   0,
			Element: nil,
			Out:     []*test{},
		},
		"negative number": {
			Times:   -5,
			Element: test1,
			Out:     nil,
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			out := funk.Repeat(test.Element, test.Times)
			for idx, el := range out {
				if el != test.Out[idx] {
					t.Logf("Pointers are not equal %p == %p", el, test.Out[idx])
					t.Fail()
				}
			}
			assert.Equal(t, test.Out, out)
		})
	}
}

func TestTail(t *testing.T) {
	tests := map[string]struct {
		input    []string
		expected []string
	}{
		"pop 1": {
			input:    []string{"1", "2", "3"},
			expected: []string{"2", "3"},
		},
		"pop none": {
			input:    []string{"1"},
			expected: []string{"1"},
		},
		"empty": {
			input:    []string{},
			expected: []string{},
		},
	}

	for name, test := range tests {
		test := test
		t.Run(name, func(t *testing.T) {
			output := funk.Tail(test.input)

			assert.Equal(t, test.expected, output)
		})
	}
}

func TestReverse(t *testing.T) {
	tests := map[string]struct {
		In  []int
		Out []int
	}{
		"odd values": {
			In:  []int{1, 2, 3, 4, 5},
			Out: []int{5, 4, 3, 2, 1},
		},
		"even values": {
			In:  []int{1, 2, 3, 4},
			Out: []int{4, 3, 2, 1},
		},
		"no values": {
			In:  []int{},
			Out: []int{},
		},
		"one value": {
			In:  []int{1},
			Out: []int{1},
		},
	}

	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			res := funk.Reverse(data.In)
			assert.Equal(t, data.Out, res)
		})
	}
}

type partitionable struct {
	name string
	time int
}

func TestPartition(t *testing.T) {
	tests := map[string]struct {
		In  []partitionable
		Out map[int][]partitionable
	}{
		"simple": {
			In: []partitionable{
				{"a", 1},
				{"b", 1},
				{"c", 2},
				{"d", 2},
				{"e", 4},
			},
			Out: map[int][]partitionable{
				1: {
					{"a", 1},
					{"b", 1},
				},
				2: {
					{"c", 2},
					{"d", 2},
				},
				4: {
					{"e", 4},
				},
			},
		},
		"empty": {
			In:  []partitionable{},
			Out: map[int][]partitionable{},
		},
		"all in one partition": {
			In: []partitionable{
				{"a", 1},
				{"b", 1},
			},
			Out: map[int][]partitionable{
				1: {
					{"a", 1},
					{"b", 1},
				},
			},
		},
	}

	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			res := funk.Partition(data.In, func(t partitionable) int {
				return t.time
			})

			assert.Equal(t, data.Out, res)
		})
	}
}

func TestUniq(t *testing.T) {
	tests := map[string]struct {
		In  []string
		Out []string
	}{
		"nil": {
			In:  nil,
			Out: []string{},
		},
		"empty": {
			In:  []string{},
			Out: []string{},
		},
		"single": {
			In:  []string{"a"},
			Out: []string{"a"},
		},
		"repeated": {
			In:  []string{"a", "a"},
			Out: []string{"a"},
		},
		"pair": {
			In:  []string{"a", "b"},
			Out: []string{"a", "b"},
		},
		"repeatedPair": {
			In:  []string{"a", "b", "a", "b"},
			Out: []string{"a", "b"},
		},
	}

	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			res := funk.Uniq(data.In)

			assert.Equal(t, data.Out, res)
		})
	}
}

func TestUniqByType(t *testing.T) {
	type A struct {
		Value int
	}
	type B struct {
		Value string
	}

	tests := map[string]struct {
		In  []any
		Out []any
	}{
		"nil": {
			In:  nil,
			Out: []any{},
		},
		"empty": {
			In:  []any{},
			Out: []any{},
		},
		"single": {
			In:  []any{&A{}},
			Out: []any{&A{}},
		},
		"repeated": {
			In:  []any{&A{}, &A{}},
			Out: []any{&A{}},
		},
		"pair": {
			In:  []any{&A{}, &B{}},
			Out: []any{&A{}, &B{}},
		},
		"repeatedPair": {
			In:  []any{&A{}, &B{}, &A{}, &B{}},
			Out: []any{&A{}, &B{}},
		},
		"repeatedPairWithValues": {
			In:  []any{&A{Value: 1}, &B{Value: "1"}, &A{Value: 2}, &B{Value: "2"}},
			Out: []any{&A{Value: 1}, &B{Value: "1"}},
		},
		"repeatedPairMixedPointers": {
			In:  []any{&A{Value: 1}, &B{Value: "1"}, A{Value: 2}, B{Value: "2"}},
			Out: []any{&A{Value: 1}, &B{Value: "1"}, A{Value: 2}, B{Value: "2"}},
		},
	}

	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			res := funk.UniqByType(data.In)

			assert.Equal(t, data.Out, res)
		})
	}
}

func TestPartitionByField(t *testing.T) {
	type A struct {
		Type  string
		Value int
	}

	tests := map[string]struct {
		In  []A
		Out map[string][]int
	}{
		"simple": {
			In: []A{
				{Type: "a", Value: 1},
				{Type: "a", Value: 2},
				{Type: "a", Value: 3},
				{Type: "b", Value: 4},
			},
			Out: map[string][]int{
				"a": {1, 2, 3},
				"b": {4},
			},
		},
		"empty": {
			In:  []A{},
			Out: map[string][]int{},
		},
		"allInOneSlice": {
			In: []A{
				{Type: "a", Value: 1},
				{Type: "a", Value: 2},
			},
			Out: map[string][]int{
				"a": {1, 2},
			},
		},
		"multipleEqualValues": {
			In: []A{
				{Type: "a", Value: 2},
				{Type: "a", Value: 2},
				{Type: "a", Value: 2},
			},
			Out: map[string][]int{
				"a": {2, 2, 2},
			},
		},
	}

	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			keyer := func(el A) (string, int) {
				return el.Type, el.Value
			}

			assert.Equal(t, data.Out, funk.PartitionMap(data.In, keyer))
		})
	}
}

func TestAny(t *testing.T) {
	tests := map[string]struct {
		In  []int
		Out bool
	}{
		"simple": {
			In:  []int{1, 2, 3},
			Out: true,
		},
		"empty": {
			In:  []int{},
			Out: false,
		},
		"all false": {
			In:  []int{0, 0, 0},
			Out: false,
		},
		"mixed": {
			In:  []int{0, 1, 0},
			Out: true,
		},
	}

	biggerThenZero := func(i int) bool {
		return i > 0
	}

	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, data.Out, funk.Any(data.In, biggerThenZero))
		})
	}
}

func TestAll(t *testing.T) {
	tests := map[string]struct {
		In  []int
		Out bool
	}{
		"simple": {
			In:  []int{1, 2, 3},
			Out: true,
		},
		"empty": {
			In:  []int{},
			Out: true,
		},
		"all false": {
			In:  []int{0, 0, 0, -1},
			Out: false,
		},
		"mixed": {
			In:  []int{0, 1, 0},
			Out: false,
		},
	}

	biggerThenZero := func(i int) bool {
		return i > 0
	}

	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, data.Out, funk.All(data.In, biggerThenZero))
		})
	}
}

func TestNone(t *testing.T) {
	tests := map[string]struct {
		In  []int
		Out bool
	}{
		"simple": {
			In:  []int{1, 2, 3},
			Out: false,
		},
		"empty": {
			In:  []int{},
			Out: true,
		},
		"all false": {
			In:  []int{0, 0, 0, -1},
			Out: true,
		},
		"mixed": {
			In:  []int{0, 1, 0},
			Out: false,
		},
	}

	biggerThenZero := func(i int) bool {
		return i > 0
	}

	for name, data := range tests {
		data := data
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, data.Out, funk.None(data.In, biggerThenZero))
		})
	}
}
