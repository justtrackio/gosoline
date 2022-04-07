package funk_test

import (
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestChunk(t *testing.T) {
	sl := []int{0, 1, 2, 3, 4, 5, 6}

	e := [][]int{
		{0, 1, 2},
		{3, 4, 5},
		{6},
	}

	res := funk.Chunk(sl, 3)
	assert.Equal(t, e, res)
}

func TestChunkReduce(t *testing.T) {
	sl := []int{0, 1, 2, 3, 4, 5, 6}

	e := [][]int{
		{0, 1, 2},
		{3, 4, 5},
		{6},
	}

	res := funk.ChunkReduce(sl, 3)
	assert.Equal(t, e, res)
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
