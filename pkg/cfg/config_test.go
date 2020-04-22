package cfg_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
	"math"
	"strings"
	"testing"
	"time"
)

var baseSettings = map[string]interface{}{
	"a": 1,
	"n": map[string]interface{}{
		"c": 3,
	},
}

func TestConfig_AllKeys(t *testing.T) {
	config := getNewTestableConfig(baseSettings, map[string]string{})
	keys := config.AllKeys()

	assert.Len(t, keys, 2)
}

func TestConfig_IsSet(t *testing.T) {
	config := getNewTestableConfig(baseSettings, map[string]string{})

	assert.True(t, config.IsSet("a"))
	assert.True(t, config.IsSet("n.c"))
	assert.False(t, config.IsSet("b"))
}

func TestConfig_Get(t *testing.T) {
	config := getNewTestableConfig(baseSettings, map[string]string{})

	expectedMap := map[string]interface{}{
		"c": 3,
	}

	assert.Equal(t, 1, config.Get("a"))
	assert.Equal(t, expectedMap, config.Get("n"))
}

func TestConfig_GetBool(t *testing.T) {
	config := getNewTestableConfig(map[string]interface{}{
		"b": true,
	}, map[string]string{})

	assert.True(t, config.GetBool("b"))
}

func TestConfig_GetDuration(t *testing.T) {
	config := getNewTestableConfig(map[string]interface{}{
		"d": time.Second,
	}, map[string]string{})

	assert.Equal(t, time.Second, config.GetDuration("d"))
}

func TestConfig_GetInt(t *testing.T) {
	config := getNewTestableConfig(map[string]interface{}{
		"i": 1,
	}, map[string]string{})

	assert.Equal(t, 1, config.GetInt("i"))
}

func TestConfig_GetIntSlice(t *testing.T) {
	config := getNewTestableConfig(map[string]interface{}{
		"slice": []int{30, 60, 120},
	}, map[string]string{})

	s := config.GetIntSlice("slice")

	assert.Len(t, s, 3)
	assert.Equal(t, 30, s[0])
	assert.Equal(t, 60, s[1])
	assert.Equal(t, 120, s[2])
}

func TestConfig_GetFloat64(t *testing.T) {
	config := getNewTestableConfig(map[string]interface{}{
		"pi": math.Pi,
	}, map[string]string{})

	assert.Equal(t, math.Pi, config.GetFloat64("pi"))
}

func TestConfig_GetString(t *testing.T) {
	config := getNewTestableConfig(map[string]interface{}{
		"s":      "foobar",
		"a":      "this {is} augmented",
		"is":     "is also {nested}",
		"nested": "nested stuff",
	}, map[string]string{})

	assert.Equal(t, "foobar", config.GetString("s"))
	assert.Equal(t, "this is also nested stuff augmented", config.GetString("a"))
}

func TestConfig_GetStringMapString(t *testing.T) {
	config := getNewTestableConfig(map[string]interface{}{
		"map": map[string]interface{}{
			"a":   "b",
			"foo": "{bar}-{map.c}",
			"c":   "d",
		},
		"bar": "baz",
	}, map[string]string{})

	strMap := config.GetStringMapString("map")
	assert.Len(t, strMap, 3)
	assert.Equal(t, "b", strMap["a"])
	assert.Equal(t, "baz-d", strMap["foo"])
}

func TestConfig_GetStringSlice(t *testing.T) {
	config := getNewTestableConfig(map[string]interface{}{
		"slice": []string{"string", "a{b}"},
		"b":     "bc",
	}, map[string]string{})

	s := config.GetStringSlice("slice")

	assert.Len(t, s, 2)
	assert.Equal(t, "string", s[0])
	assert.Equal(t, "abc", s[1])
}

func TestConfig_Environment(t *testing.T) {
	config := getNewTestableConfig(baseSettings, map[string]string{
		"A":   "2",
		"N_C": "4",
	})

	expectedMap := map[string]interface{}{
		"c": "4",
	}

	assert.Equal(t, 2, config.GetInt("a"))
	assert.Equal(t, expectedMap, config.Get("n"))
}

func TestConfig_EnvironmentPrefixed(t *testing.T) {
	config := getNewTestableConfig(baseSettings, map[string]string{
		"PREFIX_A":   "2",
		"PREFIX_N_C": "4",
	})
	_ = config.Option(cfg.WithEnvKeyPrefix("prefix"))

	expectedMap := map[string]interface{}{
		"c": "4",
	}

	assert.Equal(t, 2, config.GetInt("a"))
	assert.Equal(t, expectedMap, config.Get("n"))
}

func TestConfig_UnmarshalKey_Struct(t *testing.T) {
	type configMap struct {
		Foo          string   `cfg:"foo"`
		Bla          string   `cfg:"bla"`
		Def          int      `cfg:"def" default:"1"`
		Slice        []int    `cfg:"slice"`
		SliceAugment []string `cfg:"slice_augment"`
		Nested       struct {
			A         time.Duration `cfg:"a" default:"1s"`
			Augmented string        `cfg:"augmented"`
		} `cfg:"nested"`
	}

	config := getNewTestableConfig(map[string]interface{}{
		"key": map[string]interface{}{
			"foo":           "zorg",
			"bla":           "test",
			"slice":         []interface{}{1, 2},
			"slice_augment": []interface{}{"a", "b-{key2}"},
			"nested": map[string]interface{}{
				"augmented": "my-{key3}",
			},
		},
		"key2": "c",
		"key3": "value",
	}, map[string]string{})

	cm := configMap{}
	config.UnmarshalKey("key", &cm)

	assert.Equal(t, "zorg", cm.Foo)
	assert.Equal(t, "test", cm.Bla)
	assert.Equal(t, 1, cm.Def)
	assert.Equal(t, []int{1, 2}, cm.Slice)
	assert.Equal(t, []string{"a", "b-c"}, cm.SliceAugment)
	assert.Equal(t, time.Second, cm.Nested.A)
	assert.Equal(t, "my-value", cm.Nested.Augmented)
}

func TestConfig_UnmarshalKey_StructWithMap(t *testing.T) {
	type configMap struct {
		MSI map[string]interface{} `cfg:"msi"`
	}

	config := getNewTestableConfig(map[string]interface{}{
		"key": map[string]interface{}{
			"msi": map[string]interface{}{
				"augmented": "my-{augment}",
			},
		},
		"augment": "value",
	}, map[string]string{})

	expected := map[string]interface{}{
		"augmented": "my-value",
	}

	cm := configMap{}
	config.UnmarshalKey("key", &cm)

	assert.Equal(t, expected, cm.MSI)
}

func TestConfig_UnmarshalKey_Slice(t *testing.T) {
	type configMap struct {
		Foo string `cfg:"foo"`
		Def int    `cfg:"def" default:"1"`
	}

	config := getNewTestableConfig(map[string]interface{}{
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
	}, map[string]string{
		"KEY_2_FOO": "env",
	})

	cm := make([]configMap, 0)
	config.UnmarshalKey("key", &cm)

	assert.Len(t, cm, 3)
	assert.Equal(t, "bar", cm[0].Foo)
	assert.Equal(t, 1, cm[0].Def)
	assert.Equal(t, "baz", cm[1].Foo)
	assert.Equal(t, 2, cm[1].Def)
	assert.Equal(t, "env", cm[2].Foo)
	assert.Equal(t, 3, cm[2].Def)
}

func TestConfig_UnmarshalKeyEnvironment(t *testing.T) {
	type configMap struct {
		Foo    string `cfg:"foo"`
		Nested struct {
			A int `cfg:"a"`
		} `cfg:"nested"`
	}

	config := getNewTestableConfig(map[string]interface{}{
		"key": map[string]interface{}{
			"foo": "bar",
		},
	}, map[string]string{
		"KEY_FOO":      "zorg",
		"KEY_NESTED_A": "1",
	})

	cm := configMap{}
	config.UnmarshalKey("key", &cm)

	assert.Equal(t, "zorg", cm.Foo)
	assert.Equal(t, 1, cm.Nested.A)
}

func TestConfig_UnmarshalKeyEmbedded(t *testing.T) {
	type Embedded struct {
		A int `cfg:"a"`
		B int `cfg:"b" default:"2"`
	}

	type configMap struct {
		Embedded
		Foo string `cfg:"foo"`
	}

	config := getNewTestableConfig(map[string]interface{}{
		"key": map[string]interface{}{
			"foo": "bar",
		},
	}, map[string]string{
		"KEY_FOO": "zorg",
		"KEY_A":   "1",
	})

	cm := configMap{}
	config.UnmarshalKey("key", &cm)

	assert.Equal(t, "zorg", cm.Foo)
	assert.Equal(t, 1, cm.A)
	assert.Equal(t, 2, cm.B)
}

func TestConfig_UnmarshalKeyValidation(t *testing.T) {
	type configMap struct {
		Foo    string `cfg:"foo" validate:"oneof=baz"`
		Nested struct {
			A int `cfg:"a" validate:"gt=3"`
		} `cfg:"nested" validate:"dive"`
	}

	var cfgErr error
	errorHandler := func(err error, msg string, args ...interface{}) {
		cfgErr = err
	}

	config := getNewTestableConfigWithOptions(map[string]interface{}{
		"key": map[string]interface{}{
			"foo": "bar",
			"nested": map[string]interface{}{
				"augmented": 1,
			},
		},
	}, map[string]string{}, cfg.WithErrorHandlers(errorHandler))

	cm := configMap{}
	config.UnmarshalKey("key", &cm)

	assert.EqualError(t, cfgErr, "2 errors occurred:\n\t* the setting Foo with value bar does not match its requirement\n\t* the setting A with value 0 does not match its requirement\n\n")
}

func TestConfig_GetTime(t *testing.T) {
	data := map[string]interface{}{
		"key": map[string]interface{}{
			"foo":            "bar",
			"date":           time.Date(2019, time.November, 26, 0, 0, 0, 0, time.UTC),
			"stringDate":     "2019-11-27",
			"stringDateTime": "2020-04-22T07:17:13+02:00",
		},
	}

	config := getNewTestableConfig(data, map[string]string{})
	tm := config.GetTime("key.date")

	assert.Equal(t, "2019-11-26", tm.Format("2006-01-02"))

	settings := struct {
		Foo            string    `cfg:"foo"`
		Date           time.Time `cfg:"date"`
		StringDate     time.Time `cfg:"stringDate"`
		StringDateTime time.Time `cfg:"stringDateTime"`
	}{}

	config.UnmarshalKey("key", &settings)

	assert.Equal(t, "bar", settings.Foo)
	assert.Equal(t, "2019-11-26", settings.Date.Format("2006-01-02"))
	assert.Equal(t, "2019-11-27", settings.StringDate.Format("2006-01-02"))
	assert.Equal(t, "2020-04-22T07:17:13+02:00", settings.StringDateTime.Format(time.RFC3339))
}

func TestConfig_FromYml(t *testing.T) {
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

	config := getNewTestableConfig(map[string]interface{}{}, map[string]string{}, cfg.WithConfigFile("./testdata/config.test.yml", "yml"))

	cm := configMap{}
	config.UnmarshalKey("key", &cm)

	assert.Equal(t, 1, config.GetInt("i"))
	assert.Equal(t, expected, cm)
}

func getNewTestableConfig(settings map[string]interface{}, environment map[string]string, options ...cfg.Option) cfg.GosoConf {
	options = append(options, []cfg.Option{
		cfg.WithErrorHandlers(cfg.PanicErrorHandler),
		cfg.WithEnvKeyReplacer(strings.NewReplacer(".", "_")),
		cfg.WithSanitizers(cfg.TimeSanitizer),
	}...)

	return getNewTestableConfigWithOptions(settings, environment, options...)
}

func getNewTestableConfigWithOptions(settings map[string]interface{}, environment map[string]string, options ...cfg.Option) cfg.GosoConf {
	envMock := func(key string) (string, bool) {
		if value, ok := environment[key]; ok {
			return value, true
		}

		return "", false
	}

	options = append(options, cfg.WithConfigMap(settings))

	config := cfg.NewWithInterfaces(envMock)
	err := config.Option(options...)

	if err != nil {
		panic(err)
	}

	return config
}
