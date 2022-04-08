package funk_test

import (
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	square := func(i int) int {
		return i * i
	}

	got := funk.Map(input, square)
	assert.Equal(t, []int{1, 4, 9, 16, 25, 36, 49, 64, 81}, got)
}

func TestFilter(t *testing.T) {
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	even := func(i int) bool {
		return i%2 == 0
	}

	got := funk.Filter(input, even)
	assert.Equal(t, []int{2, 4, 6, 8}, got)
}

func TestReduce(t *testing.T) {
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	sum := func(partial int, item int, _ int) int {
		return partial + item
	}

	got := funk.Reduce(input, sum, 1000)
	assert.Equal(t, 1045, got)
}

func TestToMap(t *testing.T) {
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	keyer := func(item int) (int, int) {
		return item % 3, item / 3
	}

	got := funk.ToMap(input, keyer)

	assert.Equal(t, 3, len(got))
	assert.Contains(t, got, 0)
	assert.Contains(t, got, 1)
	assert.Contains(t, got, 2)
}

func TestToSet(t *testing.T) {
	input := []int{1, 2, 3, 1, 2, 3, 1, 2, 3}

	got := funk.ToSet(input)

	assert.Contains(t, got, 1)
	assert.Contains(t, got, 2)
	assert.Contains(t, got, 3)
	assert.NotContains(t, got, 0)
	assert.NotContains(t, got, 4)
}

func TestFindFirstFunc(t *testing.T) {
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	seven := func(item int) bool {
		return item == 7
	}

	got, ok := funk.FindFirstFunc(input, seven)

	assert.True(t, ok)
	assert.Equal(t, 6, got)
}

func TestReverse(t *testing.T) {
	type testCase struct {
		Name string
		In   []int
		Out  []int
	}

	cases := []testCase{
		{
			Name: "odd values",
			In:   []int{1, 2, 3, 4, 5},
			Out:  []int{5, 4, 3, 2, 1},
		},
		{
			Name: "even values",
			In:   []int{1, 2, 3, 4},
			Out:  []int{4, 3, 2, 1},
		},
		{
			Name: "no values",
			In:   []int{},
			Out:  []int{},
		},
		{
			Name: "one value",
			In:   []int{1},
			Out:  []int{1},
		},
	}

	for _, test := range cases {
		res := funk.Reverse(test.In)
		assert.Equalf(t, test.Out, res, "Test failed: %s", test.Name)
	}
}

func TestChunk(t *testing.T) {
	type input struct {
		Sl   []int
		Size int
	}

	type testCase struct {
		Name string
		In   input
		Out  [][]int
	}

	cases := []testCase{
		{
			Name: "remainder one",
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
		{
			Name: "remainder two",
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
		{
			Name: "size negative",
			In: input{
				Sl:   []int{0, 1, 2, 3, 4, 5, 6},
				Size: -3,
			},
			Out: nil,
		},
		{
			Name: "size zero",
			In: input{
				Sl:   []int{0, 1, 2, 3, 4, 5, 6},
				Size: 0,
			},
			Out: nil,
		},
		{
			Name: "size one",
			In: input{
				Sl:   []int{0, 1, 2, 3},
				Size: 1,
			},
			Out: [][]int{{0}, {1}, {2}, {3}},
		},
		{
			Name: "size larger than slice",
			In: input{
				Sl:   []int{0, 1, 2, 3},
				Size: 5,
			},
			Out: [][]int{{0, 1, 2, 3}},
		},
		{
			Name: "no values",
			In: input{
				Sl:   []int{},
				Size: 5,
			},
			Out: nil,
		},
	}

	for _, test := range cases {
		res := funk.Chunk(test.In.Sl, test.In.Size)
		resRed := funk.ChunkReduce(test.In.Sl, test.In.Size)
		assert.Equalf(t, test.Out, res, "Test static failed: %s", test.Name)
		assert.Equalf(t, test.Out, resRed, "Test reduce failed: %s", test.Name)
	}
}

func TestDifference(t *testing.T) {
	type input struct {
		Name string

		Input1 []int
		Input2 []int

		Out1 []int
		Out2 []int
	}

	cases := []input{
		{
			Name:   "simple",
			Input1: []int{1, 2, 3, 4},
			Input2: []int{2, 3, 5, 6},
			Out1:   []int{1, 4},
			Out2:   []int{5, 6},
		},
		{
			Name:   "identical",
			Input1: []int{1, 2, 3},
			Input2: []int{1, 2, 3},
			Out1:   []int{},
			Out2:   []int{},
		},
		{
			Name:   "disjunct",
			Input1: []int{1, 2},
			Input2: []int{3, 4},
			Out1:   []int{1, 2},
			Out2:   []int{3, 4},
		},
		{
			Name:   "left empty",
			Input1: []int{},
			Input2: []int{3, 4},
			Out1:   []int{},
			Out2:   []int{3, 4},
		},
		{
			Name:   "right empty",
			Input1: []int{1, 2},
			Input2: []int{},
			Out1:   []int{1, 2},
			Out2:   []int{},
		},
	}

	for _, test := range cases {
		l, r := funk.Difference(test.Input1, test.Input2)
		assert.ElementsMatchf(t, l, test.Out1, test.Name)
		assert.ElementsMatchf(t, r, test.Out2, test.Name)
	}
}

func TestIntersect(t *testing.T) {
	type input struct {
		Name string

		Input1 []int
		Input2 []int

		Out []int
	}

	cases := []input{
		{
			Name:   "simple",
			Input1: []int{1, 2, 3, 4},
			Input2: []int{2, 3, 5, 6},
			Out:    []int{2, 3},
		},
		{
			Name:   "identical",
			Input1: []int{1, 2, 3},
			Input2: []int{1, 2, 3},
			Out:    []int{1, 2, 3},
		},
		{
			Name:   "disjunct",
			Input1: []int{1, 2},
			Input2: []int{3, 4},
			Out:    []int{},
		},
		{
			Name:   "left empty",
			Input1: []int{},
			Input2: []int{3, 4},
			Out:    []int{},
		},
		{
			Name:   "right empty",
			Input1: []int{1, 2},
			Input2: []int{},
			Out:    []int{},
		},
	}

	for _, test := range cases {
		res := funk.Intersect(test.Input1, test.Input2)
		assert.ElementsMatchf(t, res, test.Out, "Test name: %s", test.Name)
	}
}
