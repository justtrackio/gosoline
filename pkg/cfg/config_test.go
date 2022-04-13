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

func (s *ConfigTestSuite) errorHandler(msg string, args ...interface{}) {
	s.FailNow(fmt.Errorf(msg, args...).Error())
}

func (s *ConfigTestSuite) setupConfigValues(values map[string]interface{}) {
	if err := s.config.Option(cfg.WithConfigMap(values)); err != nil {
		s.FailNow("can not setup config values", err.Error())
	}
}

func (s *ConfigTestSuite) setupEnvironment(values map[string]string) {
	for k, v := range values {
		_ = s.envProvider.SetEnv(k, v)
	}
}

func (s *ConfigTestSuite) TestConfig_AllKeys() {
	s.setupConfigValues(map[string]interface{}{
		"i": 1,
		"s": "string",
	})

	keys := s.config.AllKeys()
	s.Len(keys, 2)
}

func (s *ConfigTestSuite) TestConfig_IsSet() {
	s.setupConfigValues(map[string]interface{}{
		"i": 1,
		"ms": map[string]interface{}{
			"b": true,
		},
	})

	s.True(s.config.IsSet("i"))
	s.True(s.config.IsSet("ms.b"))
	s.False(s.config.IsSet("missing"))
}

func (s *ConfigTestSuite) TestConfig_Get() {
	s.setupConfigValues(map[string]interface{}{
		"i": 1,
		"ms": map[string]interface{}{
			"b": true,
		},
	})

	expectedMap := map[string]interface{}{
		"b": true,
	}

	s.Equal(1, s.config.Get("i"))
	s.Equal(expectedMap, s.config.Get("ms"))
	s.Equal(expectedMap, s.config.Get("ms"), map[string]interface{}{
		"c": false,
	})
	s.Equal(true, s.config.Get("missing", true))
}

func (s *ConfigTestSuite) TestConfig_GetBool() {
	s.setupConfigValues(map[string]interface{}{
		"b": "true",
	})

	s.True(s.config.GetBool("b"))
	s.True(s.config.GetBool("missing", true))
}

func (s *ConfigTestSuite) TestConfig_GetDuration() {
	s.setupConfigValues(map[string]interface{}{
		"d": "1s",
	})

	s.Equal(time.Second, s.config.GetDuration("d"))
	s.Equal(time.Minute, s.config.GetDuration("missing", time.Minute))
}

func (s *ConfigTestSuite) TestConfig_GetInt() {
	s.setupConfigValues(map[string]interface{}{
		"i": "1",
	})

	s.Equal(1, s.config.GetInt("i"))
	s.Equal(2, s.config.GetInt("missing", 2))
}

func (s *ConfigTestSuite) TestConfig_GetIntSlice() {
	s.setupConfigValues(map[string]interface{}{
		"slice": []int{30, 60, 120},
	})

	slice := s.config.GetIntSlice("slice")
	missing := s.config.GetIntSlice("missing", []int{1, 2})

	s.Equal(slice, []int{30, 60, 120})
	s.Equal(missing, []int{1, 2})
}

func (s *ConfigTestSuite) TestConfig_GetFloat64() {
	s.setupConfigValues(map[string]interface{}{
		"f64": math.Pi,
	})

	s.Equal(math.Pi, s.config.GetFloat64("f64"))
	s.Equal(math.Phi, s.config.GetFloat64("missing", math.Phi))
}

func (s *ConfigTestSuite) TestConfig_GetMsiSlice() {
	s.setupConfigValues(map[string]interface{}{
		"msi": []map[string]interface{}{
			{
				"i": 1,
				"s": "string",
			},
		},
	})

	expected := []map[string]interface{}{
		{
			"i": 1,
			"s": "string",
		},
	}

	s.Equal(expected, s.config.GetMsiSlice("msi"))
}

func (s *ConfigTestSuite) TestConfig_GetString() {
	s.setupConfigValues(map[string]interface{}{
		"s":      "foobar",
		"a":      "this {is} augmented",
		"is":     "is also {nested}",
		"nested": "nested stuff",
	})

	s.Equal("foobar", s.config.GetString("s"))
	s.Equal("this is also nested stuff augmented", s.config.GetString("a"))
	s.Equal("default", s.config.GetString("missing", "default"))
}

func (s *ConfigTestSuite) TestConfig_GetStringMapString() {
	s.setupConfigValues(map[string]interface{}{
		"map": map[string]interface{}{
			"a":   "b",
			"foo": "{bar}-{map.c}",
			"c":   "d",
		},
		"bar": "baz",
	})

	missingMap := map[string]string{
		"s": "string",
	}

	strMap := s.config.GetStringMapString("map")

	s.Len(strMap, 3)
	s.Equal("b", strMap["a"])
	s.Equal("baz-d", strMap["foo"])
	s.Equal(missingMap, s.config.GetStringMapString("missing", missingMap))
}

func (s *ConfigTestSuite) TestConfig_GetStringSlice() {
	s.setupConfigValues(map[string]interface{}{
		"slice":  []string{"string", "a{b}"},
		"b":      "bc",
		"single": "s",
		"split":  "x,y,z",
		"ints":   []interface{}{1, 2, 3},
	})

	missingSlice := []string{"a", "b"}

	ss := s.config.GetStringSlice("slice")
	single := s.config.GetStringSlice("single")
	split := s.config.GetStringSlice("split")
	ints := s.config.GetStringSlice("ints")

	s.Len(ss, 2)
	s.Equal("string", ss[0])
	s.Equal("abc", ss[1])
	s.Equal([]string{"s"}, single)
	s.Equal([]string{"x", "y", "z"}, split)
	s.Equal([]string{"1", "2", "3"}, ints)
	s.Equal(missingSlice, s.config.GetStringSlice("missing", missingSlice))
}

func (s *ConfigTestSuite) TestConfig_GetTime() {
	s.setupConfigValues(map[string]interface{}{
		"key": map[string]interface{}{
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

	s.config.UnmarshalKey("key", &settings)

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
	s.Equal("string", s.config.GetString("s"))
	s.Equal("2019-11-27", s.config.GetTime("t").Format("2006-01-02"))
	s.Equal([]string{"a", "b", "c"}, s.config.GetStringSlice("sl"))
}

func (s *ConfigTestSuite) TestConfig_EnvironmentPrefixed() {
	s.applyOptions(cfg.WithEnvKeyPrefix("prefix"))
	s.setupEnvironment(map[string]string{
		"PREFIX_I": "2",
		"PREFIX_S": "string",
	})

	s.Equal(2, s.config.GetInt("i"))
	s.Equal("string", s.config.GetString("s"))
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
	s.config.UnmarshalKey("prefix", &cm)

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

	s.setupConfigValues(map[string]interface{}{
		"key": map[string]interface{}{
			"foo":           "zorg",
			"bla":           "test",
			"slice":         []interface{}{1, 2},
			"slice_augment": []interface{}{"a", "b-{key2}"},
			"nested": map[string]interface{}{
				"augmented": "my-{key3}",
			},
		},
		"key2":  "c",
		"key3":  "value",
		"keyAD": "augmented default",
	})

	cm := configMap{}
	s.config.UnmarshalKey("key", &cm)

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
		MSI map[string]interface{} `cfg:"msi"`
	}

	s.setupConfigValues(map[string]interface{}{
		"key": map[string]interface{}{
			"msi": map[string]interface{}{
				"augmented": "my-{augment}",
			},
		},
		"augment": "value",
	})

	expected := map[string]interface{}{
		"augmented": "my-value",
	}

	cm := configMap{}
	s.config.UnmarshalKey("key", &cm)

	s.Equal(expected, cm.MSI)
}

func (s *ConfigTestSuite) TestConfig_UnmarshalKey_Slice() {
	type configMap struct {
		Foo string `cfg:"foo"`
		Def int    `cfg:"def" default:"1"`
	}

	s.setupConfigValues(map[string]interface{}{
		"key": []interface{}{
			map[string]interface{}{
				"foo": "bar",
			},
			map[string]interface{}{
				"foo": "baz",
				"def": 2,
			},
			map[string]interface{}{
				"def": 3,
			},
		},
	})
	s.setupEnvironment(map[string]string{
		"KEY_2_FOO": "env",
	})

	cm := make([]configMap, 0)
	s.config.UnmarshalKey("key", &cm)

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

	s.setupConfigValues(map[string]interface{}{
		"key": map[string]interface{}{
			"key1": map[string]interface{}{
				"foo": "bar",
			},
			"key2": map[string]interface{}{
				"foo": "baz",
				"def": 2,
			},
			"key3": map[string]interface{}{
				"def": 3,
			},
		},
	})
	s.setupEnvironment(map[string]string{
		"KEY_KEY3_FOO": "env",
	})

	cm := map[string]configMap{}
	s.config.UnmarshalKey("key", &cm)

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

	s.setupConfigValues(map[string]interface{}{
		"data": map[string]interface{}{
			"map": map[int]interface{}{
				1: "foo",
				2: "bar",
				3: "baz",
			},
		},
	})

	cm := configMap{}
	s.config.UnmarshalKey("data", &cm)

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

	s.setupConfigValues(map[string]interface{}{
		"key": map[string]interface{}{
			"foo": "bar",
		},
	})
	s.setupEnvironment(map[string]string{
		"KEY_FOO":      "zorg",
		"KEY_NESTED_A": "1",
	})

	cm := configMap{}
	s.config.UnmarshalKey("key", &cm)

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

	s.setupConfigValues(map[string]interface{}{
		"key": map[string]interface{}{
			"foo": "bar",
		},
	})
	s.setupEnvironment(map[string]string{
		"KEY_FOO": "zorg",
		"KEY_A":   "1",
	})

	cm := configMap{}
	s.config.UnmarshalKey("key", &cm)

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

	var cfgErr error
	errorHandler := func(msg string, args ...interface{}) {
		cfgErr = fmt.Errorf(msg, args...)
	}
	s.applyOptions(cfg.WithErrorHandlers(errorHandler))

	s.setupConfigValues(map[string]interface{}{
		"key": map[string]interface{}{
			"foo": "bar",
			"nested": map[string]interface{}{
				"augmented": 1,
			},
		},
	})

	cm := configMap{}
	s.config.UnmarshalKey("key", &cm)

	s.EqualError(cfgErr, "validation failed for key: key: 2 errors occurred:\n\t* the setting Foo with value bar does not match its requirement\n\t* the setting A with value 0 does not match its requirement\n\n")
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

	s.setupConfigValues(map[string]interface{}{
		"key": map[string]interface{}{
			"s": "string",
			"nested": map[string]interface{}{
				"i": 2,
			},
		},
		"additionalDefaults": map[string]interface{}{
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
	s.config.UnmarshalKey("key", &cm, cfg.UnmarshalWithDefaultsFromKey("additionalDefaults", "nested"))

	s.Equal(expected, cm)
}

func (s *ConfigTestSuite) TestConfig_FromYml() {
	type configMap struct {
		D   time.Duration          `cfg:"d"`
		I   int                    `cfg:"i"`
		MSI map[string]interface{} `cfg:"msi"`
		S1  []int                  `cfg:"s1"`
		S2  []interface{}          `cfg:"s2"`
	}

	expected := configMap{
		D: time.Minute,
		I: 2,
		MSI: map[string]interface{}{
			"s": "string",
			"d": "1s",
		},
		S1: []int{1, 2},
		S2: []interface{}{3, "s"},
	}

	s.applyOptions(cfg.WithConfigFile("./testdata/config.test.yml", "yml"))

	cm := configMap{}
	s.config.UnmarshalKey("key", &cm)

	s.Equal(1, s.config.GetInt("i"))
	s.Equal(expected, cm)
}

func TestConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}
