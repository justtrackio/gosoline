package funk_test

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/funk"
)

var (
	s = rand.NewSource(time.Now().UnixNano())
	r = rand.New(s)
)

var (
	mapResult []float64
	mapOp     = func(i int) float64 {
		return 1 / math.Sqrt(float64(i))
	}
)

func BenchmarkMap(b *testing.B) {
	var res []float64
	inp := randomIntSlice(1_000_000)

	for n := 0; n < b.N; n++ {
		res = funk.Map(inp, mapOp)
	}

	mapResult = res
}

var uniqResult []int

func BenchmarkUniq(b *testing.B) {
	var res []int
	inp := randomIntSlice(1_000_000)

	for n := 0; n < b.N; n++ {
		res = funk.Uniq(inp)
	}

	uniqResult = res
}

type testStruct struct {
	test1 int
	test3 float64
}

var uniqStructResult []testStruct

func BenchmarkUniqStruct(b *testing.B) {
	var res []testStruct
	inp := randomStructSlice(1_000_000)

	for n := 0; n < b.N; n++ {
		res = funk.Uniq(inp)
	}

	uniqStructResult = res
}

var chunkResult [][]int

func BenchmarkChunk(b *testing.B) {
	var res [][]int
	inp := randomIntSlice(1_000_000)
	for n := 0; n < b.N; n++ {
		res = funk.Chunk(inp, 100)
	}

	chunkResult = res
}

var (
	differenceResultA       []int
	differenceResultB       []int
	differenceResultAStruct []testStruct
	differenceResultBStruct []testStruct
)

func BenchmarkDifferenceRandomStruct(b *testing.B) {
	var resA, resB []int
	as := randomIntSlice(100)
	bs := randomIntSlice(100)
	for n := 0; n < b.N; n++ {
		resA, resB = funk.Difference(as, bs)
	}

	differenceResultA, differenceResultB = resA, resB
}

func BenchmarkDifferenceRandom(b *testing.B) {
	var resA, resB []testStruct
	as := randomStructSlice(100)
	bs := randomStructSlice(100)
	for n := 0; n < b.N; n++ {
		resA, resB = funk.Difference(as, bs)
	}

	differenceResultAStruct, differenceResultBStruct = resA, resB
}

func BenchmarkDifferenceStatic(b *testing.B) {
	var resA, resB []int
	as := staticIntSlice(100)
	bs := staticIntSlice(100)
	for n := 0; n < b.N; n++ {
		resA, resB = funk.Difference(as, bs)
	}

	differenceResultA, differenceResultB = resA, resB
}

var intersectResult []int

func BenchmarkIntersect(b *testing.B) {
	var res []int
	as := staticIntSlice(100)
	bs := staticIntSlice(100)
	for n := 0; n < b.N; n++ {
		res = funk.Intersect(as, bs)
	}

	intersectResult = res
}

func randomIntSlice(length int) []int {
	var res []int
	for i := 0; i < length; i++ {
		res = append(res, r.Int())
	}

	return res
}

func randomStructSlice(length int) []testStruct {
	var res []testStruct
	for i := 0; i < length; i++ {
		res = append(res, testStruct{
			test1: r.Int(),
			test3: r.Float64(),
		})
	}

	return res
}

func staticIntSlice(length int) []int {
	var a []int
	for i := 0; i < length; i++ {
		a = append(a, i)
	}

	return a
}
