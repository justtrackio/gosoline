package cfg_test

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/justtrackio/gosoline/pkg/cfg"
	"github.com/justtrackio/gosoline/pkg/clock"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
	suite.Suite

	config      cfg.GosoConf
	envProvider cfg.EnvProvider
}

func (s *ConfigTestSuite) SetupTest() {
	s.envProvider = cfg.NewMemoryEnvProvider()

	options := []cfg.Option{
		cfg.WithErrorHandlers(s.errorHandler),
		cfg.WithEnvKeyReplacer(cfg.DefaultEnvKeyReplacer),
		cfg.WithSanitizers(cfg.TimeSanitizer),
	}

	s.config = cfg.NewWithInterfaces(s.envProvider)
	s.applyOptions(options...)
}

func (s *ConfigTestSuite) applyOptions(options ...cfg.Option) {
	if err := s.config.Option(options...); err != nil {
		s.FailNow("con not apply config options", err.Error())
	}
}

func (s *ConfigTestSuite) errorHandler(msg string, args ...any) {
	s.FailNow(fmt.Errorf(msg, args...).Error())
}

func (s *ConfigTestSuite) setupConfigValues(values map[string]any) {
	if err := s.config.Option(cfg.WithConfigMap(values)); err != nil {
		s.FailNow("can not setup config values", err.Error())
	}
}

func (s *ConfigTestSuite) setupEnvironment(values map[string]string) {
	for k, v := range values {
		if err := s.envProvider.SetEnv(k, v); err != nil {
			s.FailNow("can not setup environment variable", fmt.Sprintf("key: %s, value: %s, error: %s", k, v, err.Error()))
		}
	}
}

func (s *ConfigTestSuite) TestConfig_AllKeys() {
	s.setupConfigValues(map[string]any{
		"i": 1,
		"s": "string",
	})

	keys := s.config.AllKeys()
	s.Len(keys, 2)
}

func (s *ConfigTestSuite) TestConfig_IsSet() {
	s.setupConfigValues(map[string]any{
		"i": 1,
		"ms": map[string]any{
			"b": true,
		},
	})

	s.True(s.config.IsSet("i"))
	s.True(s.config.IsSet("ms.b"))
	s.False(s.config.IsSet("missing"))
}

func (s *ConfigTestSuite) TestConfig_HasPrefix() {
	s.setupConfigValues(map[string]any{
		"i": 1,
		"ms": map[string]any{
			"b": true,
		},
	})

	s.True(s.config.HasPrefix("i"))
	s.False(s.config.HasPrefix("k"))
	s.True(s.config.HasPrefix("ms"))
	s.True(s.config.HasPrefix("ms.b"))
	s.False(s.config.HasPrefix("ms.b.c"))

	s.setupEnvironment(map[string]string{
		"FOO":          "bar",
		"NESTED_KEY_A": "1",
	})

	s.True(s.config.HasPrefix("foo"))
	s.False(s.config.HasPrefix("baz"))
	s.True(s.config.HasPrefix("nested.key"))
	s.False(s.config.HasPrefix("nested.keys"))
}

func (s *ConfigTestSuite) TestConfig_Get() {
	s.setupConfigValues(map[string]any{
		"i": 1,
		"ms": map[string]any{
			"b": true,
		},
	})

	expectedMap := map[string]any{
		"b": true,
	}

	s.Equal(1, s.config.Get("i"))
	s.Equal(expectedMap, s.config.Get("ms"))
	s.Equal(expectedMap, s.config.Get("ms"), map[string]any{
		"c": false,
	})
	s.Equal(true, s.config.Get("missing", true))
}

func (s *ConfigTestSuite) TestConfig_GetBool() {
	s.setupConfigValues(map[string]any{
		"b": "true",
	})

	val, err := s.config.GetBool("b")
	s.NoError(err)
	s.True(val)

	val, err = s.config.GetBool("missing", true)
	s.NoError(err)
	s.True(val)
}

func (s *ConfigTestSuite) TestConfig_GetDuration() {
	s.setupConfigValues(map[string]any{
		"d": "1s",
	})

	s.Equal(time.Second, s.config.GetDuration("d"))
	s.Equal(time.Minute, s.config.GetDuration("missing", time.Minute))
}

func (s *ConfigTestSuite) TestConfig_GetInt() {
	s.setupConfigValues(map[string]any{
		"i": "1",
	})

	s.Equal(1, s.config.GetInt("i"))
	s.Equal(2, s.config.GetInt("missing", 2))
}

func (s *ConfigTestSuite) TestConfig_GetIntSlice() {
	s.setupConfigValues(map[string]any{
		"slice": []int{30, 60, 120},
	})

	slice := s.config.GetIntSlice("slice")
	missing := s.config.GetIntSlice("missing", []int{1, 2})

	s.Equal(slice, []int{30, 60, 120})
	s.Equal(missing, []int{1, 2})
}

func (s *ConfigTestSuite) TestConfig_GetFloat64() {
	s.setupConfigValues(map[string]any{
		"f64": math.Pi,
	})

	s.Equal(math.Pi, s.config.GetFloat64("f64"))
	s.Equal(math.Phi, s.config.GetFloat64("missing", math.Phi))
}

func (s *ConfigTestSuite) TestConfig_GetMsiSlice() {
	s.setupConfigValues(map[string]any{
		"msi": []map[string]any{
			{
				"i": 1,
				"s": "string",
			},
		},
	})

	expected := []map[string]any{
		{
			"i": 1,
			"s": "string",
		},
	}

	s.Equal(expected, s.config.GetMsiSlice("msi"))
}

func (s *ConfigTestSuite) TestConfig_GetString() {
	s.setupConfigValues(map[string]any{
		"s":      "foobar",
		"a":      "this {is} augmented",
		"is":     "is also {nested}",
		"nested": "nested stuff",
	})

	val, err := s.config.GetString("s")
	s.NoError(err)
	s.Equal("foobar", val)

	val, err = s.config.GetString("a")
	s.NoError(err)
	s.Equal("this is also nested stuff augmented", val)

	val, err = s.config.GetString("missing", "default")
	s.NoError(err)
	s.Equal("default", val)
}

func (s *ConfigTestSuite) TestConfig_GetStringNoDecode() {
	s.setupConfigValues(map[string]any{
		"nodecode": "!nodecode {this}-should-{be}-plain",
	})

	val, err := s.config.GetString("nodecode")
	s.NoError(err)
	s.Equal("{this}-should-{be}-plain", val)
}

func (s *ConfigTestSuite) TestConfig_GetStringMapString() {
	s.setupConfigValues(map[string]any{
		"map": map[string]any{
			"a":   "b",
			"foo": "{bar}-{map.c}",
			"c":   "d",
		},
		"bar": "baz",
	})

	missingMap := map[string]string{
		"s": "string",
	}

	strMap, err := s.config.GetStringMapString("map")
	s.NoError(err)

	s.Len(strMap, 3)
	s.Equal("b", strMap["a"])
	s.Equal("baz-d", strMap["foo"])

	val, err := s.config.GetStringMapString("missing", missingMap)
	s.NoError(err)
	s.Equal(missingMap, val)
}

func (s *ConfigTestSuite) TestConfig_GetStringSlice() {
	s.setupConfigValues(map[string]any{
		"slice":  []string{"string", "a{b}"},
		"b":      "bc",
		"single": "s",
		"split":  "x,y,z",
		"ints":   []any{1, 2, 3},
	})

	missingSlice := []string{"a", "b"}

	ss, err := s.config.GetStringSlice("slice")
	s.NoError(err)

	single, err := s.config.GetStringSlice("single")
	s.NoError(err)

	split, err := s.config.GetStringSlice("split")
	s.NoError(err)

	ints, err := s.config.GetStringSlice("ints")
	s.NoError(err)

	s.Len(ss, 2)
	s.Equal("string", ss[0])
	s.Equal("abc", ss[1])
	s.Equal([]string{"s"}, single)
	s.Equal([]string{"x", "y", "z"}, split)
	s.Equal([]string{"1", "2", "3"}, ints)

	val, err := s.config.GetStringSlice("missing", missingSlice)
	s.NoError(err)
	s.Equal(missingSlice, val)
}

func (s *ConfigTestSuite) TestConfig_GetTime() {
	s.setupConfigValues(map[string]any{
		"key": map[string]any{
			"foo":            "bar",
			"date":           time.Date(2019, time.November, 26, 0, 0, 0, 0, time.UTC),
			"stringDate":     "2019-11-27",
			"stringDateTime": "2020-04-22T07:17:13+02:00",
		},
	})

	tm := s.config.GetTime("key.date")
	s.Equal("2019-11-26", tm.Format("2006-01-02"))

	settings := struct {
		Foo            string    `cfg:"foo"`
		Date           time.Time `cfg:"date"`
		StringDate     time.Time `cfg:"stringDate"`
		StringDateTime time.Time `cfg:"stringDateTime"`
	}{}
	err := s.config.UnmarshalKey("key", &settings)
	s.NoError(err)

	s.Equal("bar", settings.Foo)
	s.Equal("2019-11-26", settings.Date.Format("2006-01-02"))
	s.Equal("2019-11-27", settings.StringDate.Format("2006-01-02"))
	s.Equal("2020-04-22T07:17:13+02:00", settings.StringDateTime.Format(time.RFC3339))

	fakeTime := clock.NewFakeClock().Now()
	s.Equal(fakeTime, s.config.GetTime("missing", fakeTime))
}

func (s *ConfigTestSuite) TestConfig_Environment() {
	s.setupEnvironment(map[string]string{
		"I":  "2",
		"S":  "string",
		"T":  "2019-11-27",
		"SL": "a,b,c",
	})

	s.Equal(2, s.config.GetInt("i"))

	val, err := s.config.GetString("s")
	s.NoError(err)
	s.Equal("string", val)

	s.Equal("2019-11-27", s.config.GetTime("t").Format("2006-01-02"))

	slice, err := s.config.GetStringSlice("sl")
	s.NoError(err)
	s.Equal([]string{"a", "b", "c"}, slice)
}

func (s *ConfigTestSuite) TestConfig_EnvironmentPrefixed() {
	s.applyOptions(cfg.WithEnvKeyPrefix("prefix"))
	s.setupEnvironment(map[string]string{
		"PREFIX_I": "2",
		"PREFIX_S": "string",
	})

	s.Equal(2, s.config.GetInt("i"))

	val, err := s.config.GetString("s")
	s.NoError(err)
	s.Equal("string", val)
}

func (s *ConfigTestSuite) TestEnvironmentUnmarshalStructWithEmbeddedSlice() {
	type configMap struct {
		Slice []struct {
			S string `cfg:"s"`
		} `cfg:"slice"`
		Strings  []string `cfg:"strings"`
		Integers []int    `cfg:"integers"`
	}

	s.setupEnvironment(map[string]string{
		"PREFIX_SLICE_0_S": "a",
		"PREFIX_SLICE_1_S": "b",
		"PREFIX_STRINGS_0": "foo",
		"PREFIX_STRINGS_1": "bar",
		"PREFIX_INTEGERS":  "1,2,3",
	})

	cm := configMap{}
	err := s.config.UnmarshalKey("prefix", &cm)
	s.NoError(err)

	s.Len(cm.Slice, 2)
	s.Equal([]string{"foo", "bar"}, cm.Strings)
	s.Equal([]int{1, 2, 3}, cm.Integers)
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKey_Struct() {
	type configMap struct {
		Foo          string   `cfg:"foo"`
		Bla          string   `cfg:"bla"`
		Def          int      `cfg:"def" default:"1"`
		AugDef       string   `cfg:"aug_def" default:"{keyAD}"`
		Slice        []int    `cfg:"slice"`
		SliceAugment []string `cfg:"slice_augment"`
		Nested       struct {
			A         time.Duration `cfg:"a" default:"1s"`
			Augmented string        `cfg:"augmented"`
		} `cfg:"nested"`
	}

	s.setupConfigValues(map[string]any{
		"key": map[string]any{
			"foo":           "zorg",
			"bla":           "test",
			"slice":         []any{1, 2},
			"slice_augment": []any{"a", "b-{key2}"},
			"nested": map[string]any{
				"augmented": "my-{key3}",
			},
		},
		"key2":  "c",
		"key3":  "value",
		"keyAD": "augmented default",
	})

	cm := configMap{}
	err := s.config.UnmarshalKey("key", &cm)
	s.NoError(err)

	s.Equal("zorg", cm.Foo)
	s.Equal("test", cm.Bla)
	s.Equal(1, cm.Def)
	s.Equal("augmented default", cm.AugDef)
	s.Equal([]int{1, 2}, cm.Slice)
	s.Equal([]string{"a", "b-c"}, cm.SliceAugment)
	s.Equal(time.Second, cm.Nested.A)
	s.Equal("my-value", cm.Nested.Augmented)
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKey_StructWithMap() {
	type configMap struct {
		MSI map[string]any `cfg:"msi"`
	}

	s.setupConfigValues(map[string]any{
		"key": map[string]any{
			"msi": map[string]any{
				"augmented": "my-{augment}",
			},
		},
		"augment": "value",
	})

	expected := map[string]any{
		"augmented": "my-value",
	}

	cm := configMap{}
	err := s.config.UnmarshalKey("key", &cm)
	s.NoError(err)

	s.Equal(expected, cm.MSI)
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKey_Slice() {
	type configMap struct {
		Foo string `cfg:"foo"`
		Def int    `cfg:"def" default:"1"`
	}

	s.setupConfigValues(map[string]any{
		"key": []any{
			map[string]any{
				"foo": "bar",
			},
			map[string]any{
				"foo": "baz",
				"def": 2,
			},
			map[string]any{
				"def": 3,
			},
		},
	})
	s.setupEnvironment(map[string]string{
		"KEY_2_FOO": "env",
	})

	cm := make([]configMap, 0)
	err := s.config.UnmarshalKey("key", &cm)
	s.NoError(err)

	s.Len(cm, 3)
	s.Equal("bar", cm[0].Foo)
	s.Equal(1, cm[0].Def)
	s.Equal("baz", cm[1].Foo)
	s.Equal(2, cm[1].Def)
	s.Equal("env", cm[2].Foo)
	s.Equal(3, cm[2].Def)
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKey_Map() {
	type configMap struct {
		Foo string `cfg:"foo"`
		Def int    `cfg:"def" default:"1"`
	}

	s.setupConfigValues(map[string]any{
		"key": map[string]any{
			"key1": map[string]any{
				"foo": "bar",
			},
			"key2": map[string]any{
				"foo": "baz",
				"def": 2,
			},
			"key3": map[string]any{
				"def": 3,
			},
		},
	})
	s.setupEnvironment(map[string]string{
		"KEY_KEY3_FOO": "env",
	})

	cm := map[string]configMap{}
	err := s.config.UnmarshalKey("key", &cm)
	s.NoError(err)

	s.Len(cm, 3)
	s.Contains(cm, "key1")
	s.Contains(cm, "key2")
	s.Contains(cm, "key3")
	s.Equal(configMap{Foo: "bar", Def: 1}, cm["key1"])
	s.Equal(configMap{Foo: "baz", Def: 2}, cm["key2"])
	s.Equal(configMap{Foo: "env", Def: 3}, cm["key3"])
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKey_MapWithIntKeys() {
	type configMap struct {
		Map map[uint]string `cfg:"map"`
	}

	s.setupConfigValues(map[string]any{
		"data": map[string]any{
			"map": map[int]any{
				1: "foo",
				2: "bar",
				3: "baz",
			},
		},
	})

	cm := configMap{}
	err := s.config.UnmarshalKey("data", &cm)
	s.NoError(err)

	s.Equal(configMap{
		Map: map[uint]string{
			1: "foo",
			2: "bar",
			3: "baz",
		},
	}, cm)
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKeyEnvironment() {
	type configMap struct {
		Foo    string `cfg:"foo"`
		Nested struct {
			A int `cfg:"a"`
		} `cfg:"nested"`
	}

	s.setupConfigValues(map[string]any{
		"key": map[string]any{
			"foo": "bar",
		},
	})
	s.setupEnvironment(map[string]string{
		"KEY_FOO":      "zorg",
		"KEY_NESTED_A": "1",
	})

	cm := configMap{}
	err := s.config.UnmarshalKey("key", &cm)
	s.NoError(err)

	s.Equal("zorg", cm.Foo)
	s.Equal(1, cm.Nested.A)
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKeyEmbedded() {
	type Embedded struct {
		A int `cfg:"a"`
		B int `cfg:"b" default:"2"`
	}

	type configMap struct {
		Embedded
		Foo string `cfg:"foo"`
	}

	s.setupConfigValues(map[string]any{
		"key": map[string]any{
			"foo": "bar",
		},
	})
	s.setupEnvironment(map[string]string{
		"KEY_FOO": "zorg",
		"KEY_A":   "1",
	})

	cm := configMap{}
	err := s.config.UnmarshalKey("key", &cm)
	s.NoError(err)

	s.Equal("zorg", cm.Foo)
	s.Equal(1, cm.A)
	s.Equal(2, cm.B)
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKeyValidation() {
	type configMap struct {
		Foo    string `cfg:"foo" validate:"oneof=baz"`
		Nested struct {
			A int `cfg:"a" validate:"gt=3"`
		} `cfg:"nested" validate:"dive"`
	}

	s.setupConfigValues(map[string]any{
		"mykey": map[string]any{
			"foo": "bar",
			"nested": map[string]any{
				"augmented": 1,
			},
		},
	})

	cm := configMap{}
	err := s.config.UnmarshalKey("mykey", &cm)
	s.Require().Error(err)
	s.EqualError(err, "can not unmarshal config struct with key mykey: validation failed for key: mykey: 2 errors occurred:\n\t* the setting Foo with value bar does not match its requirement\n\t* the setting A with value 0 does not match its requirement\n\n")
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKeyWithDefaultsFromKey() {
	type ConfigNested struct {
		I int  `cfg:"i" default:"1"`
		B bool `cfg:"b"`
	}

	type configMap struct {
		S      string       `cfg:"s"`
		Nested ConfigNested `cfg:"nested"`
	}

	s.setupConfigValues(map[string]any{
		"key": map[string]any{
			"s": "string",
			"nested": map[string]any{
				"i": 2,
			},
		},
		"additionalDefaults": map[string]any{
			"i": 3,
			"b": true,
		},
	})

	expected := configMap{
		S: "string",
		Nested: ConfigNested{
			I: 2,
			B: true,
		},
	}

	cm := configMap{}
	err := s.config.UnmarshalKey("key", &cm, cfg.UnmarshalWithDefaultsFromKey("additionalDefaults", "nested"))
	s.NoError(err)

	s.Equal(expected, cm)
}

func (s *ConfigTestSuite) TestConfig_FromYml() {
	type configMap struct {
		D   time.Duration  `cfg:"d"`
		I   int            `cfg:"i"`
		MSI map[string]any `cfg:"msi"`
		S1  []int          `cfg:"s1"`
		S2  []any          `cfg:"s2"`
	}

	expected := configMap{
		D: time.Minute,
		I: 2,
		MSI: map[string]any{
			"s": "string",
			"d": "1s",
		},
		S1: []int{1, 2},
		S2: []any{3, "s"},
	}

	s.applyOptions(cfg.WithConfigFile("./testdata/config.test.yml", "yml"))

	cm := configMap{}
	err := s.config.UnmarshalKey("key", &cm)
	s.NoError(err)

	s.Equal(1, s.config.GetInt("i"))
	s.Equal(expected, cm)
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKey_Defaults() {
	type configMap struct {
		Foo          string    `cfg:"foo" default:"fooVal"`
		Bar          int       `cfg:"bar" default:"123"`
		Baz          bool      `cfg:"baz" default:"true"`
		Baz2         bool      `cfg:"baz2" default:"false"`
		StringSlice  []string  `cfg:"string_slice" default:"a,b,c"`
		String       string    `cfg:"string_slice" default:"a,b,c"`
		IntSlice     []int     `cfg:"int_slice" default:"-1,0,1"`
		Int64Slice   []int64   `cfg:"int64_slice" default:"-9223372036854775808,0,9223372036854775807"`
		Float32Slice []float32 `cfg:"float32_slice" default:"1.234,2.345,3.456"`
		Float64Slice []float64 `cfg:"float64_slice" default:"1.234,2.345,3.456"`
		BoolSlice    []bool    `cfg:"bool_slice" default:"true,false,true"`
	}

	cm := configMap{}
	err := s.config.UnmarshalKey("key", &cm)
	s.NoError(err)

	s.Equal("fooVal", cm.Foo)
	s.Equal(123, cm.Bar)
	s.Equal(true, cm.Baz)
	s.Equal(false, cm.Baz2)
	s.Equal([]string{"a", "b", "c"}, cm.StringSlice)
	s.Equal("a,b,c", cm.String)
	s.Equal([]int{-1, 0, 1}, cm.IntSlice)
	s.Equal([]int64{-9223372036854775808, 0, 9223372036854775807}, cm.Int64Slice)
	s.Equal([]float32{1.234, 2.345, 3.456}, cm.Float32Slice)
	s.Equal([]float64{1.234, 2.345, 3.456}, cm.Float64Slice)
	s.Equal([]bool{true, false, true}, cm.BoolSlice)
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
