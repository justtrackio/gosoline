package cfg_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"math"
	"testing"
	"time"
)

type testType struct {
	String   string        `cfg:"string"`
	Bool     bool          `cfg:"bool"`
	Int      int           `cfg:"int"`
	Duration time.Duration `cfg:"duration"`
	Slice    []string      `cfg:"slice"`
	Float64  float64       `cfg:"float64"`
}

func TestNew(t *testing.T) {
	viper := getViper()

	_ = ioutil.WriteFile("./config.dist.yml", nil, 0777)
	_ = cfg.NewWithInterfaces(nil, viper, "app")

	viper.AssertExpectations(t)
}

func TestConfig_AllKeys(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("AllKeys").Return([]string{"test", "test2"})

	keys := config.AllKeys()

	assert.Len(t, keys, 2)

	viper.AssertExpectations(t)
}

func TestConfig_IsSet(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "string").Return(true)

	exists := config.IsSet("string")
	assert.Equal(t, true, exists)

	viper.AssertExpectations(t)
}

func TestConfig_Get(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "string").Return(true)
	viper.On("Get", "string").Return("string")

	s := config.Get("string")
	assert.Equal(t, "string", s)

	viper.AssertExpectations(t)
}

func TestConfig_GetDuration(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "duration").Return(true)
	viper.On("GetDuration", "duration").Return(time.Duration(2))

	d := config.GetDuration("duration")
	assert.Equal(t, time.Duration(2), d)

	viper.AssertExpectations(t)
}

func TestConfig_GetInt(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "int").Return(true)
	viper.On("GetInt", "int").Return(1)

	i := config.GetInt("int")
	assert.Equal(t, 1, i)

	viper.AssertExpectations(t)
}

func TestConfig_GetFloat64(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "float64").Return(true)
	viper.On("GetFloat64", "float64").Return(math.Pi)

	i := config.GetFloat64("float64")
	assert.Equal(t, math.Pi, i)

	viper.AssertExpectations(t)
}

func TestConfig_GetString(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "string").Return(true)
	viper.On("GetString", "string").Return("string")

	s := config.GetString("string")
	assert.Equal(t, "string", s)

	viper.AssertExpectations(t)
}

func TestConfig_GetStringSlice(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "string").Return(true)
	viper.On("GetStringSlice", "string").Return([]string{"string"})

	s := config.GetStringSlice("string")
	assert.Equal(t, []string{"string"}, s)

	viper.AssertExpectations(t)
}

func TestConfig_GetBool(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "bool").Return(true)
	viper.On("GetBool", "bool").Return(true)

	b := config.GetBool("bool")
	assert.Equal(t, true, b)

	viper.AssertExpectations(t)
}

func TestConfig_Unmarshal(t *testing.T) {
	type configMap struct {
		Foo       string `cfg:"foo"`
		Bla       string `cfg:"bla"`
		Augmented string `cfg:"augmented"`
	}

	config, viper := getNewConfig()

	viper.On("IsSet", "foo").Return(true)
	viper.On("Get", "foo").Return("bar")
	viper.On("AllSettings").Return(map[string]interface{}{
		"foo":       "bar",
		"bla":       "test",
		"augmented": "{foo}-baz",
	})

	cm := configMap{}
	config.Unmarshal(&cm)

	assert.Equal(t, "bar", cm.Foo)
	assert.Equal(t, "test", cm.Bla)
	assert.Equal(t, "bar-baz", cm.Augmented)

	viper.AssertExpectations(t)
}

func TestConfig_UnmarshalKey(t *testing.T) {
	type configMap struct {
		Foo string `cfg:"foo"`
		Bla string `cfg:"bla"`
	}

	config, viper := getNewConfig()

	viper.On("IsSet", "env").Return(true)
	viper.On("Get", "env").Return("test")

	viper.On("IsSet", "key").Return(true)
	viper.On("Get", "key").Return(map[string]interface{}{
		"foo": "bar",
		"bla": "{env}",
	})

	viper.On("IsSet", "key.foo").Return(true)
	viper.On("Get", "key.foo").Return("zorg")

	viper.On("IsSet", "key.bla").Return(true)
	viper.On("Get", "key.bla").Return("{env}")

	cm := configMap{}
	config.UnmarshalKey("key", &cm)

	assert.Equal(t, "zorg", cm.Foo)
	assert.Equal(t, "test", cm.Bla)

	viper.AssertExpectations(t)
}

func TestConfig_AugmentString(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "bar").Return(true)
	viper.On("Get", "bar").Return("baz")

	str := "foo-{bar}-baz"
	str = config.AugmentString(str)

	assert.Equal(t, "foo-baz-baz", str)
}

func getViper() *cfgMocks.Viper {
	viper := new(cfgMocks.Viper)
	viper.On("SetEnvPrefix", "app")
	viper.On("AutomaticEnv")
	viper.On("SetEnvKeyReplacer", mock.AnythingOfType("*strings.Replacer"))
	viper.On("SetConfigType", "yml")
	viper.On("MergeConfig", mock.AnythingOfType("*os.File")).Return(nil)
	viper.On("GetString", "env").Return("test")

	return viper
}

func getNewConfig() (cfg.Config, *cfgMocks.Viper) {
	viper := getViper()

	_ = ioutil.WriteFile("./config.dist.yml", nil, 0777)

	config := cfg.NewWithInterfaces(nil, viper, "app")

	return config, viper
}
