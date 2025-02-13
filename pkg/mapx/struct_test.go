package mapx_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/stretchr/testify/assert"
)

func TestNewMapStructIONoPointer(t *testing.T) {
	source := struct{}{}

	_, err := mapx.NewStruct(source, &mapx.StructSettings{})
	assert.EqualError(t, err, "the target value has to be a pointer")
}

func TestMapStructIO_KeysBasic(t *testing.T) {
	type EmbeddedA struct {
		I int `cfg:"i"`
	}

	type SlStruct struct {
		S string `cfg:"s"`
	}

	type sourceStruct struct {
		EmbeddedA
		B    bool        `cfg:"b"`
		SlSt []SlStruct  `cfg:"sl_sr"`
		SlS  []string    `cfg:"sl_s"`
		T    []time.Time `cfg:"t"`
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	keys := ms.Keys()

	assert.Len(t, keys, 5)
}

// This tesst assumes that we can pass non-pointers of the same type for writing pointers to a mapStruct
// (env vars config values to ptr struct properties case)
func TestMapStructIO_PointerTarget(t *testing.T) {
	type sourceStruct struct {
		B   *bool           `cfg:"b"`
		D   *time.Duration  `cfg:"d"`
		MSI *map[string]any `cfg:"msi"`
		S   *string         `cfg:"s"`
		SlS *[]string       `cfg:"sl_s"`
		T   *time.Time      `cfg:"t"`
	}

	now := time.Now()

	mx := mapx.NewMapX(map[string]any{
		"b": true,
		"d": "1m",
		"msi": map[string]any{
			"foo": "bar",
		},
		"s":    "foo",
		"sl_s": []string{"a", "b"},
		"t":    now,
	})

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)

	err := ms.Write(mx)
	assert.Nil(t, err)

	assert.Equal(t, true, *source.B)
	assert.Equal(t, time.Minute, *source.D)
	assert.Equal(t, "bar", (*source.MSI)["foo"])
	assert.Equal(t, "foo", *source.S)
	assert.Equal(t, []string{"a", "b"}, *source.SlS)
	assert.Equal(t, now, *source.T)
}

func TestMapStructIO_ReadZeroAndDefaultValuesBasic(t *testing.T) {
	type sourceStruct struct {
		B    bool          `cfg:"b" default:"true"`
		D    time.Duration `cfg:"d" default:"1s"`
		I    int           `cfg:"i" default:"1"`
		I8   int8          `cfg:"i8" default:"2"`
		I16  int16         `cfg:"i16" default:"3"`
		I32  int32         `cfg:"i32" default:"4"`
		I64  int64         `cfg:"i64" default:"5"`
		F32  float32       `cfg:"f32" default:"1.1"`
		F64  float64       `cfg:"f64" default:"1.2"`
		S    string        `cfg:"s" default:"string"`
		T    time.Time     `cfg:"t" default:"2020-04-21"`
		UI   uint          `cfg:"ui" default:"1"`
		UI8  uint8         `cfg:"ui8" default:"2"`
		UI16 uint16        `cfg:"ui16" default:"3"`
		UI32 uint32        `cfg:"ui32" default:"4"`
		UI64 uint64        `cfg:"ui64" default:"5"`
	}

	source := &sourceStruct{}

	ms := setupMapStructIO(t, source)
	zero, defaults, err := ms.ReadZeroAndDefaultValues()
	assert.NoError(t, err, "there should be no error during reading of zeros and defaults")

	assert.Equal(t, false, zero.Get("b").Data())
	assert.Equal(t, true, defaults.Get("b").Data())
	assert.Equal(t, time.Duration(0), zero.Get("d").Data())
	assert.Equal(t, time.Second, defaults.Get("d").Data())
	assert.Equal(t, 0, zero.Get("i").Data())
	assert.Equal(t, 1, defaults.Get("i").Data())
	assert.Equal(t, int8(0), zero.Get("i8").Data())
	assert.Equal(t, int8(2), defaults.Get("i8").Data())
	assert.Equal(t, int16(0), zero.Get("i16").Data())
	assert.Equal(t, int16(3), defaults.Get("i16").Data())
	assert.Equal(t, int32(0), zero.Get("i32").Data())
	assert.Equal(t, int32(4), defaults.Get("i32").Data())
	assert.Equal(t, int64(0), zero.Get("i64").Data())
	assert.Equal(t, int64(5), defaults.Get("i64").Data())
	assert.Equal(t, float32(0), zero.Get("f32").Data())
	assert.Equal(t, float32(1.1), defaults.Get("f32").Data())
	assert.Equal(t, float64(0), zero.Get("f64").Data())
	assert.Equal(t, 1.2, defaults.Get("f64").Data())
	assert.Equal(t, "", zero.Get("s").Data())
	assert.Equal(t, "string", defaults.Get("s").Data())
	assert.Equal(t, time.Time{}, zero.Get("t").Data())
	assert.Equal(t, time.Date(2020, time.April, 21, 0, 0, 0, 0, time.UTC), defaults.Get("t").Data())
	assert.Equal(t, uint(0), zero.Get("ui").Data())
	assert.Equal(t, uint(1), defaults.Get("ui").Data())
	assert.Equal(t, uint8(0), zero.Get("ui8").Data())
	assert.Equal(t, uint8(2), defaults.Get("ui8").Data())
	assert.Equal(t, uint16(0), zero.Get("ui16").Data())
	assert.Equal(t, uint16(3), defaults.Get("ui16").Data())
	assert.Equal(t, uint32(0), zero.Get("ui32").Data())
	assert.Equal(t, uint32(4), defaults.Get("ui32").Data())
	assert.Equal(t, uint64(0), zero.Get("ui64").Data())
	assert.Equal(t, uint64(5), defaults.Get("ui64").Data())
}

func TestMapStructIO_ReadZeroAndDefaultValuesMapSlice(t *testing.T) {
	type sourceStruct struct {
		Slice []string        `cfg:"slice"`
		Map   map[int]float64 `cfg:"map"`
	}

	source := &sourceStruct{}

	ms := setupMapStructIO(t, source)
	zero, defaults, err := ms.ReadZeroAndDefaultValues()
	assert.NoError(t, err, "there should be no error during reading of zeros and defaults")

	assert.Equal(t, []any{}, zero.Get("slice").Data())
	assert.False(t, defaults.Has("slice"))
	assert.Equal(t, map[int]float64{}, zero.Get("map").Data())
	assert.False(t, defaults.Has("map"))
}

func TestMapStructIO_ReadZeroAndDefaultValuesNested(t *testing.T) {
	type NestedA struct {
		I int `cfg:"i" default:"1"`
	}

	type NestedB struct {
		B      bool    `cfg:"b" default:"true"`
		Nested NestedA `cfg:"nestedA"`
	}

	type sourceStruct struct {
		S      string  `cfg:"s" default:"string"`
		Nested NestedB `cfg:"nestedB"`
	}

	source := &sourceStruct{}

	ms := setupMapStructIO(t, source)
	zero, defaults, err := ms.ReadZeroAndDefaultValues()
	assert.NoError(t, err, "there should be no error during reading of zeros and defaults")

	assert.Equal(t, "", zero.Get("s").Data())
	assert.Equal(t, "string", defaults.Get("s").Data())
	assert.Equal(t, false, zero.Get("nestedB.b").Data())
	assert.Equal(t, true, defaults.Get("nestedB.b").Data())
	assert.Equal(t, 0, zero.Get("nestedB.nestedA.i").Data())
	assert.Equal(t, 1, defaults.Get("nestedB.nestedA.i").Data())
}

func TestMapStructIO_ReadZeroValuesAndDefaultEmbedded(t *testing.T) {
	type EmbeddedA struct {
		I int `cfg:"i" default:"1"`
	}

	type EmbeddedB struct {
		B bool `cfg:"b" default:"true"`
		EmbeddedA
	}

	type sourceStruct struct {
		S string `cfg:"s" default:"string"`
		EmbeddedB
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	zero, defaults, err := ms.ReadZeroAndDefaultValues()
	assert.NoError(t, err, "there should be no error during reading of zeros and defaults")

	assert.Equal(t, "", zero.Get("s").Data())
	assert.Equal(t, "string", defaults.Get("s").Data())
	assert.Equal(t, 0, zero.Get("i").Data())
	assert.Equal(t, 1, defaults.Get("i").Data())
	assert.Equal(t, false, zero.Get("b").Data())
	assert.Equal(t, true, defaults.Get("b").Data())
}

func TestMapStructIO_ReadZeroAndDefaultValues_Unexported(t *testing.T) {
	type embeddedA struct {
		I int `cfg:"i" default:"1"`
	}

	type sourceStruct struct {
		S string `cfg:"s" default:"string"`
		embeddedA
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	_, _, err := ms.ReadZeroAndDefaultValues()
	assert.NoError(t, err, "there should be no error during reading of zeros and defaults")
}

func TestMapStruct_ReadBasic(t *testing.T) {
	fakeTime := clock.NewFakeClock().Now()

	type sourceStruct struct {
		B    bool              `cfg:"b"`
		D    time.Duration     `cfg:"d"`
		I    int               `cfg:"i"`
		I8   int8              `cfg:"i8"`
		I16  int16             `cfg:"i16"`
		I32  int32             `cfg:"i32"`
		I64  int64             `cfg:"i64"`
		F32  float32           `cfg:"f32"`
		F64  float64           `cfg:"f64"`
		S    string            `cfg:"s"`
		T    time.Time         `cfg:"t"`
		UI   uint              `cfg:"ui"`
		UI8  uint8             `cfg:"ui8"`
		UI16 uint16            `cfg:"ui16"`
		UI32 uint32            `cfg:"ui32"`
		UI64 uint64            `cfg:"ui64"`
		MSI  map[string]any    `cfg:"msi"`
		MSS  map[string]string `cfg:"mss"`
	}

	source := &sourceStruct{
		B:    true,
		D:    time.Second,
		I:    1,
		I8:   2,
		I16:  3,
		I32:  4,
		I64:  5,
		F32:  1.1,
		F64:  1.2,
		S:    "string",
		T:    fakeTime,
		UI:   1,
		UI8:  2,
		UI16: 3,
		UI32: 4,
		UI64: 5,
		MSI: map[string]any{
			"a": "a",
			"1": 1,
		},
		MSS: map[string]string{
			"b": "b",
			"c": "c",
		},
	}

	expectedValues := map[string]any{
		"b":    true,
		"d":    time.Second,
		"i":    1,
		"i8":   int8(2),
		"i16":  int16(3),
		"i32":  int32(4),
		"i64":  int64(5),
		"f32":  float32(1.1),
		"f64":  1.2,
		"s":    "string",
		"t":    fakeTime,
		"ui":   uint(1),
		"ui8":  uint8(2),
		"ui16": uint16(3),
		"ui32": uint32(4),
		"ui64": uint64(5),
		"msi": map[string]any{
			"a": "a",
			"1": 1,
		},
		"mss": map[string]any{
			"b": "b",
			"c": "c",
		},
	}

	ms := setupMapStructIO(t, source)
	msi, err := ms.Read()

	assert.NoError(t, err, "there should be no error during reading")
	assert.Equal(t, expectedValues, msi.Msi())
}

func TestMapStructIO_ReadNested(t *testing.T) {
	type sourceStructNested struct {
		B bool   `cfg:"b"`
		S string `cfg:"s"`
	}

	type sourceStruct struct {
		Nested sourceStructNested `cfg:"nested"`
	}

	source := &sourceStruct{
		Nested: sourceStructNested{
			B: true,
			S: "string",
		},
	}

	expectedValues := map[string]any{
		"nested": map[string]any{
			"b": true,
			"s": "string",
		},
	}

	ms := setupMapStructIO(t, source)
	msi, err := ms.Read()

	assert.NoError(t, err, "there should be no error during reading")
	assert.Equal(t, expectedValues, msi.Msi())
}

func TestMapStructIO_ReadAnonymous(t *testing.T) {
	type SourceStructAnonymous struct {
		B bool   `cfg:"b"`
		S string `cfg:"s"`
	}

	type sourceStruct struct {
		SourceStructAnonymous
	}

	source := &sourceStruct{
		SourceStructAnonymous: SourceStructAnonymous{
			B: true,
			S: "string",
		},
	}

	expectedValues := map[string]any{
		"b": true,
		"s": "string",
	}

	ms := setupMapStructIO(t, source)
	msi, err := ms.Read()

	assert.NoError(t, err, "there should be no error during reading")
	assert.Equal(t, expectedValues, msi.Msi())
}

func TestMapStructIO_ReadMapStruct(t *testing.T) {
	type SourceMapStruct struct {
		S string `cfg:"s"`
	}

	type sourceStruct struct {
		M map[string]SourceMapStruct `cfg:"m"`
	}

	source := &sourceStruct{
		M: map[string]SourceMapStruct{
			"a": {
				S: "string",
			},
		},
	}

	expectedValues := map[string]any{
		"m": map[string]any{
			"a": map[string]any{
				"s": "string",
			},
		},
	}

	ms := setupMapStructIO(t, source)
	msi, err := ms.Read()

	assert.NoError(t, err, "there should be no error during reading")
	assert.Equal(t, expectedValues, msi.Msi())
}

func TestMapStructIO_ReadSliceStruct(t *testing.T) {
	type SourceStructSlice struct {
		B bool   `cfg:"b"`
		S string `cfg:"s"`
	}

	type sourceStruct struct {
		SL []SourceStructSlice `cfg:"sl"`
	}

	source := &sourceStruct{
		SL: []SourceStructSlice{
			{
				B: false,
				S: "s1",
			},
			{
				B: true,
				S: "s2",
			},
		},
	}

	expectedValues := map[string]any{
		"sl": []any{
			map[string]any{
				"b": false,
				"s": "s1",
			},
			map[string]any{
				"b": true,
				"s": "s2",
			},
		},
	}

	ms := setupMapStructIO(t, source)
	msi, err := ms.Read()

	assert.NoError(t, err, "there should be no error during reading")
	assert.Equal(t, expectedValues, msi.Msi())
}

func TestNewMapStructIO_ReadSliceMap(t *testing.T) {
	type sourceStruct struct {
		SL []map[string]any `cfg:"sl"`
	}

	source := &sourceStruct{
		SL: []map[string]any{
			{
				"b": true,
				"i": 3,
			},
			{
				"s": "string",
				"f": 1.2,
			},
		},
	}

	expectedValues := map[string]any{
		"sl": []any{
			map[string]any{
				"b": true,
				"i": 3,
			},
			map[string]any{
				"s": "string",
				"f": 1.2,
			},
		},
	}

	ms := setupMapStructIO(t, source)
	msi, err := ms.Read()

	assert.NoError(t, err, "there should be no error during reading")
	assert.Equal(t, expectedValues, msi.Msi())
}

func TestNewMapStructIO_ReadSliceBasic(t *testing.T) {
	type sourceStruct struct {
		SL []int `cfg:"sl"`
	}

	source := &sourceStruct{
		SL: []int{1, 2, 3},
	}

	expectedValues := map[string]any{
		"sl": []any{1, 2, 3},
	}

	ms := setupMapStructIO(t, source)
	msi, err := ms.Read()

	assert.NoError(t, err, "there should be no error during reading")
	assert.Equal(t, expectedValues, msi.Msi())
}

func TestMapStructIO_WriteBasic(t *testing.T) {
	type sourceStruct struct {
		B    bool          `cfg:"b" default:"true"`
		D    time.Duration `cfg:"d" default:"1s"`
		I    int           `cfg:"i" default:"1"`
		I8   int8          `cfg:"i8" default:"2"`
		I16  int16         `cfg:"i16" default:"3"`
		I32  int32         `cfg:"i32" default:"4"`
		I64  int64         `cfg:"i64" default:"5"`
		F32  float32       `cfg:"f32" default:"1.1"`
		F64  float64       `cfg:"f64" default:"1.2"`
		S    string        `cfg:"s" default:"string"`
		T    time.Time     `cfg:"t" default:"2020-04-21"`
		UI   uint          `cfg:"ui" default:"1"`
		UI8  uint8         `cfg:"ui8" default:"2"`
		UI16 uint16        `cfg:"ui16" default:"3"`
		UI32 uint32        `cfg:"ui32" default:"4"`
		UI64 uint64        `cfg:"ui64" default:"5"`
	}

	values := mapx.NewMapX(map[string]any{
		"b":    true,
		"d":    "1s",
		"i":    1,
		"i8":   2,
		"i16":  3,
		"i32":  4,
		"i64":  5,
		"f32":  1.1,
		"f64":  1.2,
		"s":    "string",
		"t":    "2020-04-21",
		"ui":   1,
		"ui8":  2,
		"ui16": 3,
		"ui32": 4,
		"ui64": 5,
	})

	expected := &sourceStruct{
		B:    true,
		D:    time.Second,
		I:    1,
		I8:   2,
		I16:  3,
		I32:  4,
		I64:  5,
		F32:  1.1,
		F64:  1.2,
		S:    "string",
		T:    time.Date(2020, time.April, 21, 0, 0, 0, 0, time.UTC),
		UI:   1,
		UI8:  2,
		UI16: 3,
		UI32: 4,
		UI64: 5,
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStructIO_WriteEmbedded(t *testing.T) {
	type EmbeddedStruct struct {
		S string `cfg:"s"`
	}

	type sourceStruct struct {
		B bool `cfg:"b"`
		I int  `cfg:"i"`
		EmbeddedStruct
	}

	values := mapx.NewMapX(map[string]any{
		"b": "true",
		"i": 1,
		"s": "string",
	})

	expected := &sourceStruct{
		B: true,
		I: 1,
		EmbeddedStruct: EmbeddedStruct{
			S: "string",
		},
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStructIO_WriteStructNested(t *testing.T) {
	type nestedStruct struct {
		S string `cfg:"s"`
	}

	type sourceStruct struct {
		B      bool         `cfg:"b"`
		I      int          `cfg:"i"`
		Nested nestedStruct `cfg:"nested"`
	}

	values := mapx.NewMapX(map[string]any{
		"b": "true",
		"i": 1,
		"nested": map[string]any{
			"s": "string",
		},
	})

	expected := &sourceStruct{
		B: true,
		I: 1,
		Nested: nestedStruct{
			S: "string",
		},
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStructIO_WriteStructMerge(t *testing.T) {
	type nestedStruct struct {
		I int     `cfg:"i"`
		S string  `cfg:"s"`
		F float32 `cfg:"f"`
	}

	type sourceStruct struct {
		Nested nestedStruct `cfg:"nested"`
	}

	values := mapx.NewMapX(map[string]any{
		"nested": map[string]any{
			"s": "foo",
			"f": 3.0,
		},
	})

	expected := &sourceStruct{
		Nested: nestedStruct{
			I: 1,
			S: "foo",
			F: 3,
		},
	}

	source := &sourceStruct{
		Nested: nestedStruct{
			I: 1,
			F: 2,
		},
	}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStructIO_WriteZero(t *testing.T) {
	type sourceStruct struct {
		MSI map[string]any `cfg:"msi"`
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)

	zero, _, err := ms.ReadZeroAndDefaultValues()
	assert.NoError(t, err, "there should be no error during reading zero and defaults")

	err = ms.Write(zero)

	expected := &sourceStruct{
		MSI: map[string]any{},
	}

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStructIO_WriteSliceMap(t *testing.T) {
	type slice struct {
		I int `cfg:"i"`
	}

	type sourceStruct struct {
		MI  map[string]int   `cfg:"mi"`
		MS1 map[string]slice `cfg:"ms1"`
		SI  []int            `cfg:"si"`
		SS  []slice          `cfg:"ss"`
	}

	values := mapx.NewMapX(map[string]any{
		"mi": map[string]any{
			"a": 1,
			"b": 2,
		},
		"ms1": map[string]any{
			"a": map[string]any{
				"i": 1,
			},
			"b": map[string]any{
				"i": 2,
			},
		},
		"si": []any{1, 2},
		"ss": []any{
			map[string]any{
				"i": 1,
			},
			map[string]any{
				"i": 2,
			},
		},
	})

	expected := &sourceStruct{
		MI: map[string]int{
			"a": 1,
			"b": 2,
		},
		MS1: map[string]slice{
			"a": {
				I: 1,
			},
			"b": {
				I: 2,
			},
		},
		SI: []int{1, 2},
		SS: []slice{
			{
				I: 1,
			},
			{
				I: 2,
			},
		},
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStruct_Write_Basic_To_Slice(t *testing.T) {
	type sourceStruct struct {
		S  []any    `cfg:"s"`
		SB []bool   `cfg:"sb"`
		SI []int    `cfg:"si"`
		SS []string `cfg:"ss"`
	}

	values := mapx.NewMapX(map[string]any{
		"s":  "1,a",
		"si": 1,
		"ss": "1 ,2, a",
	})

	expected := &sourceStruct{
		S:  []any{"1", "a"},
		SI: []int{1},
		SS: []string{"1", "2", "a"},
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStruct_Write_Typed(t *testing.T) {
	type someString string
	type someStruct struct {
		Value someString `cfg:"value"`
	}

	values := mapx.NewMapX(map[string]any{
		"value": "string",
	})

	expected := &someStruct{
		Value: "string",
	}

	source := &someStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStruct_Write_Typed_Slice(t *testing.T) {
	type someString string
	type someStruct struct {
		Values []someString `cfg:"values"`
	}

	values := mapx.NewMapX(map[string]any{
		"values": []string{"string", "other string"},
	})

	expected := &someStruct{
		Values: []someString{"string", "other string"},
	}

	source := &someStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStruct_Write_Typed_StringMap(t *testing.T) {
	type someString string
	type someKey string
	type someStruct struct {
		Values map[someKey]someString `cfg:"values"`
	}

	values := mapx.NewMapX(map[string]any{
		"values": map[string]any{
			"1": "string",
			"2": "other string",
		},
	})

	expected := &someStruct{
		Values: map[someKey]someString{
			"1": "string",
			"2": "other string",
		},
	}

	source := &someStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStruct_Write_Typed_IntMap(t *testing.T) {
	type someString string
	type someInt int
	type someStruct struct {
		Values map[someInt]someString `cfg:"values"`
	}

	values := mapx.NewMapX(map[string]any{
		"values": map[string]any{
			"1": "string",
			"2": "other string",
		},
	})

	expected := &someStruct{
		Values: map[someInt]someString{
			1: "string",
			2: "other string",
		},
	}

	source := &someStruct{}
	ms := setupMapStructIO(t, source)

	// Right now this doesn't work because of the way `reflect.Value.String()` works.
	// We have this test to be notified if this changes.
	assert.Panics(t, func() {
		err := ms.Write(values)

		assert.NoError(t, err, "there should be no error during write")
		assert.Equal(t, expected, source)
	})
}

func TestMapStruct_Write_Typed_MapNestedInSlice(t *testing.T) {
	type someString string
	type someKey string
	type someStruct struct {
		Values map[someKey][]map[someString][]map[string][]someKey `cfg:"values"`
	}

	values := mapx.NewMapX(map[string]any{
		"values": map[string]any{
			"1": []any{
				map[string]any{
					"2": []any{
						map[string]any{
							"3": []any{"string", "some string"},
						},
					},
				},
			},
		},
	})

	expected := &someStruct{
		Values: map[someKey][]map[someString][]map[string][]someKey{
			"1": {
				{
					"2": []map[string][]someKey{
						{
							"3": []someKey{"string", "some string"},
						},
					},
				},
			},
		},
	}

	source := &someStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStruct_Write_Typed_MapNested(t *testing.T) {
	type someString string
	type someKey string
	type someStruct struct {
		Values map[someKey]map[someString][]map[string][]someKey `cfg:"values"`
	}

	values := mapx.NewMapX(map[string]any{
		"values": map[string]any{
			"1": map[string]any{
				"2": []any{
					map[string]any{
						"3": []any{"string", "some string"},
					},
				},
			},
		},
	})

	expected := &someStruct{
		Values: map[someKey]map[someString][]map[string][]someKey{
			"1": {
				"2": []map[string][]someKey{
					{
						"3": []someKey{"string", "some string"},
					},
				},
			},
		},
	}

	source := &someStruct{}
	ms := setupMapStructIO(t, source)

	// Right now this doesn't work because in a map there needs to be either a struct or a slice, having a map in a map
	// causes a map to be treated as a struct, which fails.
	// We have this test to be notified if this changes.
	assert.Panics(t, func() {
		err := ms.Write(values)

		assert.NoError(t, err, "there should be no error during write")
		assert.Equal(t, expected, source)
	})
}

func TestMapStruct_Decode(t *testing.T) {
	type sourceStruct struct {
		A string `cfg:"a"`
		B string `cfg:"b,nodecode"`
	}

	source := &sourceStruct{}
	ms, err := mapx.NewStruct(source, &mapx.StructSettings{
		FieldTag: "cfg",
		Decoders: []mapx.MapStructDecoder{
			func(targetType reflect.Type, val any) (any, error) {
				if raw, ok := val.(string); ok {
					return strings.ToUpper(raw), nil
				}

				return val, nil
			},
		},
	})

	assert.NoError(t, err, "there should be no error on creating the mapstruct")

	err = ms.Write(mapx.NewMapX(map[string]any{
		"a": "foo",
		"b": "bar",
	}))

	expected := &sourceStruct{
		A: "FOO",
		B: "bar",
	}

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func setupMapStructIO(t *testing.T, source any) *mapx.Struct {
	ms, err := mapx.NewStruct(source, &mapx.StructSettings{
		FieldTag:   "cfg",
		DefaultTag: "default",
		Casters: []mapx.MapStructCaster{
			mapx.MapStructDurationCaster,
			mapx.MapStructTimeCaster,
		},
	})

	assert.NoError(t, err, "there should be no error on creating the mapstruct")

	return ms
}
