package cfg_test

import (
	"github.com/applike/gosoline/pkg/cfg"
	cfgMocks "github.com/applike/gosoline/pkg/cfg/mocks"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
	"time"
)

type testType struct {
	String   string        `cfg:"string"`
	Bool     bool          `cfg:"bool"`
	Int      int           `cfg:"int"`
	Duration time.Duration `cfg:"duration"`
	Slice    []string      `cfg:"slice"`
}

func TestNew(t *testing.T) {
	viper := getViper()

	_ = ioutil.WriteFile("./config.dist.yml", nil, 0777)
	_ = cfg.New(nil, viper, "app")

	viper.AssertExpectations(t)
}

func TestConfig_AllKeys(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("AllKeys").Return([]string{"test", "test2"})

	keys := config.AllKeys()

	assert.Len(t, keys, 2)

	viper.AssertExpectations(t)
}

func TestConfig_Bind(t *testing.T) {
	config, viper := getNewConfig()

	viper.On("IsSet", "string").Return(true)
	viper.On("IsSet", "bool").Return(true)
	viper.On("IsSet", "int").Return(true)
	viper.On("IsSet", "duration").Return(true)
	viper.On("IsSet", "slice").Return(true)
	viper.On("Get", "string").Return("string")
	viper.On("GetString", "bool").Return("True")
	viper.On("GetInt", "int").Return(1)
	viper.On("GetDuration", "duration").Return(time.Duration(2))
	viper.On("Get", "slice").Return([]string{"slice"})

	obj := testType{}
	config.Bind(&obj)
	assert.Equal(t, "string", obj.String)
	assert.Equal(t, true, obj.Bool)
	assert.Equal(t, 1, obj.Int)
	assert.Equal(t, time.Duration(2), obj.Duration)
	assert.Equal(t, []string{"slice"}, obj.Slice)

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
		Foo string `mapstructure:"foo"`
		Bla string `mapstructure:"bla"`
	}

	config, viper := getNewConfig()

	viper.On("IsSet", "env").Return(true)
	viper.On("Get", "env").Return("test")

	viper.On("IsSet", "key").Return(true)
	viper.On("Get", "key").Return(map[string]interface{}{
		"foo": "bar",
		"bla": "{env}",
	})

	cm := configMap{}
	config.Unmarshal("key", &cm)

	assert.Equal(t, "bar", cm.Foo)
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
	viper.On("GetString", "env").Return("test")

	return viper
}

func getNewConfig() (cfg.Config, *cfgMocks.Viper) {
	viper := getViper()

	_ = ioutil.WriteFile("./config.dist.yml", nil, 0777)

	config := cfg.New(nil, viper, "app")

	return config, viper
}
