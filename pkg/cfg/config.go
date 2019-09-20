package cfg

import (
	"flag"
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"
)

type ConfigFlags map[string]string

func (f *ConfigFlags) String() string {
	return "my string representation"
}

func (f *ConfigFlags) Set(value string) error {
	parts := strings.Split(value, "=")
	(*f)[parts[0]] = parts[1]

	return nil
}

type DecoderConfigOption func(*mapstructure.DecoderConfig)

//go:generate mockery -name Config
type Config interface {
	AllKeys() []string
	Get(string) interface{}
	GetDuration(string) time.Duration
	GetInt(string) int
	GetFloat64(string) float64
	GetString(string) string
	GetStringMapString(key string) map[string]string
	GetStringSlice(key string) []string
	GetBool(key string) bool
	IsSet(string) bool
	Unmarshal(output interface{}, opts ...DecoderConfigOption)
	UnmarshalKey(key string, val interface{}, opts ...DecoderConfigOption)
	AugmentString(str string) string
}

type config struct {
	application string
	client      Viper
	sentry      *raven.Client
	lck         sync.Mutex
}

//go:generate mockery -name Viper
type Viper interface {
	AddConfigPath(string)
	AllKeys() []string
	AllSettings() map[string]interface{}
	AutomaticEnv()
	Get(string) interface{}
	GetBool(key string) bool
	GetDuration(string) time.Duration
	GetInt(string) int
	GetFloat64(string) float64
	GetString(string) string
	GetStringMapString(key string) map[string]string
	GetStringSlice(key string) []string
	IsSet(string) bool
	SetConfigType(in string)
	MergeConfig(in io.Reader) error
	SetDefault(string, interface{})
	SetEnvPrefix(string)
	Set(key string, value interface{})
}

func New(application string) *config {
	sentry := raven.DefaultClient
	client := viper.GetViper()

	return NewWithInterfaces(sentry, client, application)
}

func NewWithInterfaces(sentry *raven.Client, client Viper, application string) *config {
	c := &config{
		application: application,
		client:      client,
		sentry:      sentry,
	}

	c.configure()

	return c
}

func (c *config) keyCheck(key string) {
	if !c.client.IsSet(key) {
		panic(fmt.Errorf("there is no value configured for key '%v'", key))
	}
}

func (c *config) AllKeys() []string {
	c.lck.Lock()
	defer c.lck.Unlock()

	return c.client.AllKeys()
}

func (c *config) IsSet(key string) bool {
	return c.client.IsSet(key)
}

func (c *config) Get(key string) interface{} {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)
	return c.client.Get(key)
}

func (c *config) unsafeGet(key string) interface{} {
	c.keyCheck(key)
	return c.client.Get(key)
}

func (c *config) GetDuration(key string) time.Duration {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)
	return c.client.GetDuration(key)
}

func (c *config) GetInt(key string) int {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)
	return c.client.GetInt(key)
}

func (c *config) GetFloat64(key string) float64 {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)
	return c.client.GetFloat64(key)
}

func (c *config) GetString(key string) string {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)
	str := c.client.GetString(key)

	return c.unsafeAugmentString(str)
}

func (c *config) GetStringMapString(key string) map[string]string {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)
	configMap := c.client.GetStringMapString(key)

	for k, v := range configMap {
		configMap[k] = c.unsafeAugmentString(v)
	}

	return configMap
}

func (c *config) GetStringSlice(key string) []string {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)

	strs := c.client.GetStringSlice(key)
	for i := 0; i < len(strs); i++ {
		strs[i] = c.unsafeAugmentString(strs[i])
	}

	return strs
}

func (c *config) GetBool(key string) bool {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)
	return c.client.GetBool(key)
}

func (c *config) Unmarshal(output interface{}, opts ...DecoderConfigOption) {
	c.lck.Lock()
	defer c.lck.Unlock()

	input := c.client.AllSettings()
	err := c.decode(input, output)

	if err != nil {
		c.err(err, "error unmarshalling")
		return
	}
}

func (c *config) UnmarshalKey(key string, output interface{}, opts ...DecoderConfigOption) {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)
	input := c.unsafeGet(key)

	err := c.decode(input, output)

	if err != nil {
		c.err(err, "error unmarshalling key: %s", key)
		return
	}
}

func (c *config) decode(input interface{}, output interface{}) error {
	decoderConfig := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		TagName:          "cfg",
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			c.decodeAugmentHook(),
		),
	}

	decoder, err := mapstructure.NewDecoder(decoderConfig)

	if err != nil {
		return errors.Wrap(err, "can not initialize decoder")
	}

	err = decoder.Decode(input)

	if err != nil {
		return errors.Wrap(err, "can not decode input")
	}

	return nil
}

func (c *config) decodeAugmentHook() interface{} {
	return func(
		f reflect.Kind,
		t reflect.Kind,
		data interface{}) (interface{}, error) {
		if f != reflect.String || t != reflect.String {
			return data, nil
		}

		raw := data.(string)
		return c.unsafeAugmentString(raw), nil
	}
}

func (c *config) AugmentString(str string) string {
	c.lck.Lock()
	defer c.lck.Unlock()

	return c.unsafeAugmentString(str)
}

func (c *config) unsafeAugmentString(str string) string {
	rp := regexp.MustCompile("{([\\w]+)}")
	matches := rp.FindAllStringSubmatch(str, -1)

	for _, m := range matches {
		replace := fmt.Sprint(c.unsafeGet(m[1]))
		str = strings.Replace(str, m[0], replace, -1)
	}

	return str
}

func (c *config) configure() {
	var err error

	prefix := strings.Replace(c.application, "-", "_", -1)

	c.client.SetEnvPrefix(prefix)
	c.client.AutomaticEnv()
	c.client.SetConfigType("yml")

	err = c.readConfigFile("./config.dist.yml")

	if err != nil {
		c.err(err, "could not read default config file './config.dist.yml")
		return
	}

	if c.client.GetString("env") == "test" {
		return
	}

	flags := flag.NewFlagSet("cfg", flag.ContinueOnError)

	configFile := flags.String("config", "", "path to a config file")
	configFlags := make(ConfigFlags, 0)

	flags.Var(&configFlags, "c", "cli flags")
	_ = flags.Parse(os.Args[1:])

	err = c.readConfigFile(*configFile)

	if err != nil {
		c.err(err, "could not read the provided config file: %s", *configFile)
		return
	}

	for k, v := range configFlags {
		c.client.Set(k, v)
	}
}

func (c *config) readConfigFile(configFile string) error {
	if configFile == "" {
		return nil
	}

	file, err := os.Open(configFile)

	if err != nil {
		return err
	}

	err = c.client.MergeConfig(file)

	return err
}

func (c *config) err(err error, msg string, args ...interface{}) {
	err = errors.Wrapf(err, msg, args...)

	c.sentry.CaptureErrorAndWait(err, nil)
	panic(err)
}
