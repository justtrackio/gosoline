package cfg_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/jonboulle/clockwork"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewMapStructIONoPointer(t *testing.T) {
	source := struct {
	}{}

	_, err := cfg.NewMapStruct(source, &cfg.MapStructSettings{})
	assert.EqualError(t, err, "the target value has to be a pointer")
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

	assert.Equal(t, false, zero["b"])
	assert.Equal(t, true, defaults["b"])
	assert.Equal(t, time.Duration(0), zero["d"])
	assert.Equal(t, time.Second, defaults["d"])
	assert.Equal(t, 0, zero["i"])
	assert.Equal(t, 1, defaults["i"])
	assert.Equal(t, int8(0), zero["i8"])
	assert.Equal(t, int8(2), defaults["i8"])
	assert.Equal(t, int16(0), zero["i16"])
	assert.Equal(t, int16(3), defaults["i16"])
	assert.Equal(t, int32(0), zero["i32"])
	assert.Equal(t, int32(4), defaults["i32"])
	assert.Equal(t, int64(0), zero["i64"])
	assert.Equal(t, int64(5), defaults["i64"])
	assert.Equal(t, float32(0), zero["f32"])
	assert.Equal(t, float32(1.1), defaults["f32"])
	assert.Equal(t, float64(0), zero["f64"])
	assert.Equal(t, 1.2, defaults["f64"])
	assert.Equal(t, "", zero["s"])
	assert.Equal(t, "string", defaults["s"])
	assert.Equal(t, time.Time{}, zero["t"])
	assert.Equal(t, time.Date(2020, time.April, 21, 0, 0, 0, 0, time.UTC), defaults["t"])
	assert.Equal(t, uint(0), zero["ui"])
	assert.Equal(t, uint(1), defaults["ui"])
	assert.Equal(t, uint8(0), zero["ui8"])
	assert.Equal(t, uint8(2), defaults["ui8"])
	assert.Equal(t, uint16(0), zero["ui16"])
	assert.Equal(t, uint16(3), defaults["ui16"])
	assert.Equal(t, uint32(0), zero["ui32"])
	assert.Equal(t, uint32(4), defaults["ui32"])
	assert.Equal(t, uint64(0), zero["ui64"])
	assert.Equal(t, uint64(5), defaults["ui64"])
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

	assert.Equal(t, []string{}, zero["slice"])
	assert.NotContains(t, defaults, "slice")
	assert.Equal(t, map[int]float64{}, zero["map"])
	assert.NotContains(t, defaults, "map")
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

	assert.Equal(t, "", zero["s"])
	assert.Equal(t, "string", defaults["s"])
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

	assert.Equal(t, "", zero["s"])
	assert.Equal(t, "string", defaults["s"])
	assert.Equal(t, 0, zero["i"])
	assert.Equal(t, 1, defaults["i"])
	assert.Equal(t, false, zero["b"])
	assert.Equal(t, true, defaults["b"])
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
	fakeTime := clockwork.NewFakeClock().Now()

	type sourceStruct struct {
		B    bool          `cfg:"b"`
		D    time.Duration `cfg:"d"`
		I    int           `cfg:"i"`
		I8   int8          `cfg:"i8"`
		I16  int16         `cfg:"i16"`
		I32  int32         `cfg:"i32"`
		I64  int64         `cfg:"i64"`
		F32  float32       `cfg:"f32"`
		F64  float64       `cfg:"f64"`
		S    string        `cfg:"s"`
		T    time.Time     `cfg:"t"`
		UI   uint          `cfg:"ui"`
		UI8  uint8         `cfg:"ui8"`
		UI16 uint16        `cfg:"ui16"`
		UI32 uint32        `cfg:"ui32"`
		UI64 uint64        `cfg:"ui64"`
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
	}

	expectedValues := map[string]interface{}{
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

	expectedValues := map[string]interface{}{
		"nested": map[string]interface{}{
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

	expectedValues := map[string]interface{}{
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

	expectedValues := map[string]interface{}{
		"m": map[string]interface{}{
			"a": map[string]interface{}{
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

	expectedValues := map[string]interface{}{
		"sl": []interface{}{
			map[string]interface{}{
				"b": false,
				"s": "s1",
			},
			map[string]interface{}{
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
		SL []map[string]interface{} `cfg:"sl"`
	}

	source := &sourceStruct{
		SL: []map[string]interface{}{
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

	expectedValues := map[string]interface{}{
		"sl": []interface{}{
			map[string]interface{}{
				"b": true,
				"i": 3,
			},
			map[string]interface{}{
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

	expectedValues := map[string]interface{}{
		"sl": []interface{}{1, 2, 3},
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

	values := map[string]interface{}{
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
	}

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

	values := map[string]interface{}{
		"b": "true",
		"i": 1,
		"s": "string",
	}

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

	values := map[string]interface{}{
		"b": "true",
		"i": 1,
		"nested": map[string]interface{}{
			"s": "string",
		},
	}

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

	values := map[string]interface{}{
		"mi": map[string]interface{}{
			"a": 1,
			"b": 2,
		},
		"ms1": map[string]interface{}{
			"a": map[string]interface{}{
				"i": 1,
			},
			"b": map[string]interface{}{
				"i": 2,
			},
		},
		"si": []interface{}{1, 2},
		"ss": []interface{}{
			map[string]interface{}{
				"i": 1,
			},
			map[string]interface{}{
				"i": 2,
			},
		},
		"sm": []interface{}{
			map[string]interface{}{
				"t": "true",
				"f": "false",
			},
		},
	}

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
		//SM: []map[string]bool{
		//	{
		//		"t": true,
		//	},
		//	{
		//		"f": true,
		//	},
		//},
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func TestMapStruct_Write_Basic_To_Slice(t *testing.T) {
	type sourceStruct struct {
		S  []interface{} `cfg:"s"`
		SB []bool        `cfg:"sb"`
		SI []int         `cfg:"si"`
		SS []string      `cfg:"ss"`
	}

	values := map[string]interface{}{
		"s":  "1,a",
		"si": 1,
		"ss": "1 ,2, a",
	}

	expected := &sourceStruct{
		S:  []interface{}{"1", "a"},
		SI: []int{1},
		SS: []string{"1", "2", "a"},
	}

	source := &sourceStruct{}
	ms := setupMapStructIO(t, source)
	err := ms.Write(values)

	assert.NoError(t, err, "there should be no error during write")
	assert.Equal(t, expected, source)
}

func setupMapStructIO(t *testing.T, source interface{}) *cfg.MapStruct {
	ms, err := cfg.NewMapStruct(source, &cfg.MapStructSettings{
		FieldTag:   "cfg",
		DefaultTag: "default",
		Casters: []cfg.MapStructCaster{
			cfg.MapStructDurationCaster,
			cfg.MapStructSliceCaster,
			cfg.MapStructTimeCaster,
		},
	})

	assert.NoError(t, err, "there should be no error on creating the mapstruct")

	return ms
}
