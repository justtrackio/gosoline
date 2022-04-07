package funk_test

import (
	"github.com/justtrackio/gosoline/pkg/funk"
	goFunk "github.com/thoas/go-funk"
	"math/rand"
	"testing"
	"time"
)

var uniqResult []int

var (
	s = rand.NewSource(time.Now().UnixNano())
	r = rand.New(s)
)

func BenchmarkUniq(b *testing.B) {
	var res []int
	inp := randomIntSlice(1_000_000)

	for n := 0; n < b.N; n++ {
		res = funk.Uniq(inp)
	}

	uniqResult = res
}

func BenchmarkUniqThoas(b *testing.B) {
	var res []int
	inp := randomIntSlice(1_000_000)

	for n := 0; n < b.N; n++ {
		res = goFunk.Uniq(inp).([]int)
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

func BenchmarkUniqThoasStruct(b *testing.B) {
	var res []testStruct
	inp := randomStructSlice(1_000_000)

	for n := 0; n < b.N; n++ {
		res = goFunk.Uniq(inp).([]testStruct)
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

func BenchmarkChunkReduce(b *testing.B) {
	var res [][]int
	inp := randomIntSlice(1_000_000)
	for n := 0; n < b.N; n++ {
		res = funk.ChunkReduce(inp, 100)
	}

	chunkResult = res
}

func BenchmarkChunkThoas(b *testing.B) {
	var res [][]int
	inp := randomIntSlice(1_000_000)
	for n := 0; n < b.N; n++ {
		res = goFunk.Chunk(inp, 100).([][]int)
	}

	chunkResult = res
}

var differenceResultA []int
var differenceResultB []int
var differenceResultAStruct []testStruct
var differenceResultBStruct []testStruct

func BenchmarkDifferenceRandomStruct(b *testing.B) {
	var resA, resB []int
	as := randomIntSlice(100)
	bs := randomIntSlice(100)
	for n := 0; n < b.N; n++ {
		resA, resB = funk.Difference(as, bs)
	}

	differenceResultA, differenceResultB = resA, resB
}

func BenchmarkDifferenceThoasRandomStruct(b *testing.B) {
	var resA, resB interface{}
	as := randomStructSlice(100)
	bs := randomStructSlice(100)
	for n := 0; n < b.N; n++ {
		resA, resB = goFunk.Difference(as, bs)
	}

	differenceResultAStruct = resA.([]testStruct)
	differenceResultBStruct = resB.([]testStruct)
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

func BenchmarkDifferenceThoasRandom(b *testing.B) {
	var resA, resB interface{}
	as := randomIntSlice(100)
	bs := randomIntSlice(100)
	for n := 0; n < b.N; n++ {
		resA, resB = goFunk.Difference(as, bs)
	}

	differenceResultA = resA.([]int)
	differenceResultB = resB.([]int)
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

func BenchmarkDifferenceThoasStatic(b *testing.B) {
	var resA, resB interface{}
	as := staticIntSlice(100)
	bs := staticIntSlice(100)
	for n := 0; n < b.N; n++ {
		resA, resB = goFunk.Difference(as, bs)
	}

	differenceResultA = resA.([]int)
	differenceResultB = resB.([]int)
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

func BenchmarkIntersectThoas(b *testing.B) {
	var res []int
	as := staticIntSlice(100)
	bs := staticIntSlice(100)
	for n := 0; n < b.N; n++ {
		res = goFunk.Join(as, bs, goFunk.InnerJoin).([]int)
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
