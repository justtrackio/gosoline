package cfg_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	"github.com/stretchr/testify/assert"
	"math"
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

//func TestConfig_Unmarshal(t *testing.T) {
//	type configMap struct {
//		Foo       string `cfg:"foo"`
//		Bla       string `cfg:"bla"`
//		Augmented string `cfg:"augmented"`
//	}
//
//	config, viper := getNewTestableConfig()
//
//	viper.On("IsSet", "foo").Return(true)
//	viper.On("Get", "foo").Return("bar")
//	viper.On("AllSettings").Return(map[string]interface{}{
//		"foo":       "bar",
//		"bla":       "test",
//		"augmented": "{foo}-baz",
//	})
//
//	cm := configMap{}
//	config.Unmarshal(&cm)
//
//	assert.Equal(t, "bar", cm.Foo)
//	assert.Equal(t, "test", cm.Bla)
//	assert.Equal(t, "bar-baz", cm.Augmented)
//
//	viper.AssertExpectations(t)
//}
//
func TestConfig_UnmarshalKey(t *testing.T) {
	type configMap struct {
		Foo    string `cfg:"foo"`
		Bla    string `cfg:"bla"`
		Def    int    `cfg:"def" default:"1"`
		Nested struct {
			A         time.Duration `cfg:"a" default:"1s"`
			Augmented string        `cfg:"augmented"`
		} `cfg:"nested"`
	}

	config := getNewTestableConfig(map[string]interface{}{
		"key": map[string]interface{}{
			"foo": "zorg",
			"bla": "test",
			"nested": map[string]interface{}{
				"augmented": "my-{key2}",
			},
		},
		"key2": "value",
	}, map[string]string{})

	cm := configMap{}
	config.UnmarshalKey("key", &cm)

	assert.Equal(t, "zorg", cm.Foo)
	assert.Equal(t, "test", cm.Bla)
	assert.Equal(t, 1, cm.Def)
	assert.Equal(t, time.Second, cm.Nested.A)
	assert.Equal(t, "my-value", cm.Nested.Augmented)
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

func getNewTestableConfig(settings map[string]interface{}, environment map[string]string) cfg.GosoConf {
	options := []cfg.Option{
		cfg.WithErrorHandlers(cfg.PanicErrorHandler),
	}

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
