package cfg

import (
	"flag"
	"fmt"
	"github.com/getsentry/raven-go"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"strconv"
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

//go:generate mockery -name Config
type Config interface {
	AllKeys() []string
	Bind(obj interface{})
	Get(string) interface{}
	GetDuration(string) time.Duration
	GetInt(string) int
	GetString(string) string
	GetStringMapString(key string) map[string]string
	GetStringSlice(key string) []string
	GetBool(key string) bool
	Unmarshal(key string, val interface{})
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
	AutomaticEnv()
	Get(string) interface{}
	GetBool(key string) bool
	GetDuration(string) time.Duration
	GetInt(string) int
	GetString(string) string
	GetStringMapString(key string) map[string]string
	GetStringSlice(key string) []string
	IsSet(string) bool
	ReadInConfig() error
	SetConfigName(string)
	SetDefault(string, interface{})
	SetEnvPrefix(string)
	UnmarshalKey(string, interface{}, ...viper.DecoderConfigOption) error
	Set(key string, value interface{})
}

func New(sentry *raven.Client, client Viper, application string) *config {
	c := &config{
		application: application,
		client:      client,
		sentry:      sentry,
	}

	c.configure()

	return c
}

func NewWithDefaultClients(application string) *config {
	sentry := raven.DefaultClient
	client := viper.GetViper()

	return New(sentry, client, application)
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

func (c *config) Bind(obj interface{}) {
	r := reflect.ValueOf(obj).Elem()

	for i := 0; i < r.NumField(); i++ {
		valueField := r.Field(i)
		typeField := r.Type().Field(i)

		name := typeField.Name
		key := typeField.Tag.Get("cfg")

		if key == "" {
			panic(fmt.Errorf("there is no 'cfg' tag set for config field '%v'", name))
		}

		c.lck.Lock()
		c.keyCheck(key)
		c.lck.Unlock()

		switch typeField.Type.String() {
		case "time.Duration":
			value := c.GetDuration(key)
			valueField.Set(reflect.ValueOf(value))
		case "bool":
			value := c.GetString(key)
			boolValue, err := strconv.ParseBool(value)

			if err != nil {
				panic(err)
			}

			valueField.Set(reflect.ValueOf(boolValue))
		case "int":
			value := c.GetInt(key)
			valueField.Set(reflect.ValueOf(value))
		default:
			value := c.Get(key)
			valueField.Set(reflect.ValueOf(value))
		}
	}
}

func (c *config) Get(key string) interface{} {
	c.lck.Lock()
	defer c.lck.Unlock()

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

func (c *config) Unmarshal(key string, val interface{}) {
	c.lck.Lock()
	defer c.lck.Unlock()

	c.keyCheck(key)
	err := c.client.UnmarshalKey(key, val)

	if err != nil {
		panic(err)
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
		replace := fmt.Sprint(c.client.Get(m[1]))
		str = strings.Replace(str, m[0], replace, -1)
	}

	return str
}

func (c *config) configure() {
	var err error

	prefix := strings.Replace(c.application, "-", "_", -1)

	c.client.SetEnvPrefix(prefix)
	c.client.AutomaticEnv()

	err = c.readConfigDefault()

	if err != nil {
		c.sentry.CaptureErrorAndWait(err, nil)
		os.Exit(1)
	}

	if c.client.GetString("env") == "test" {
		return
	}

	flags := flag.NewFlagSet("cfg", flag.ContinueOnError)

	configFile := flags.String("config", "", "path to a config file")
	configFlags := make(ConfigFlags, 0)

	flags.Var(&configFlags, "c", "cli flags")
	_ = flags.Parse(os.Args[1:])

	err = c.readConfigFile(configFile)

	if err != nil {
		fmt.Printf("could not read the provided config file: %s\n", err.Error())
		c.sentry.CaptureErrorAndWait(err, nil)
		os.Exit(1)
	}

	for k, v := range configFlags {
		c.client.Set(k, v)
	}
}

func (c *config) readConfigDefault() error {
	data, err := ioutil.ReadFile("./config.dist.yml")

	if err != nil {
		fmt.Printf("could not read default config file './config.dist.yml: %s\n'", err.Error())
		return err
	}

	defaultMap := make(map[string]interface{})
	err = yaml.Unmarshal(data, &defaultMap)

	if err != nil {
		return err
	}

	for key, value := range defaultMap {
		c.client.SetDefault(key, value)
	}

	return nil
}

func (c *config) readConfigFile(configFile *string) error {
	if len(*configFile) == 0 {
		return nil
	}

	configName := *configFile
	index := strings.LastIndexAny(configName, ".")

	c.client.SetConfigName(configName[0:index]) // name of config file (without extension)
	c.client.AddConfigPath(".")                 // optionally look for config in the working directory

	err := c.client.ReadInConfig() // Find and read the config file

	return err
}
