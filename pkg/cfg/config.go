package cfg

import (
	"fmt"
	"github.com/applike/gosoline/pkg/refl"
	"github.com/hashicorp/go-multierror"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/thoas/go-funk"
	"gopkg.in/go-playground/validator.v9"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"
)

type LookupEnv func(key string) (string, bool)

//go:generate mockery -name Config
type Config interface {
	AllKeys() []string
	AllSettings() map[string]interface{}
	Get(key string, optionalDefault ...interface{}) interface{}
	GetBool(key string, optionalDefault ...bool) bool
	GetDuration(key string, optionalDefault ...time.Duration) time.Duration
	GetInt(key string, optionalDefault ...int) int
	GetIntSlice(key string, optionalDefault ...[]int) []int
	GetFloat64(key string, optionalDefault ...float64) float64
	GetMsiSlice(key string, optionalDefault ...[]map[string]interface{}) []map[string]interface{}
	GetString(key string, optionalDefault ...string) string
	GetStringMap(key string, optionalDefault ...map[string]interface{}) map[string]interface{}
	GetStringMapString(key string, optionalDefault ...map[string]string) map[string]string
	GetStringSlice(key string, optionalDefault ...[]string) []string
	GetTime(key string, optionalDefault ...time.Time) time.Time
	IsSet(string) bool
	UnmarshalDefaults(val interface{}, additionalDefaults ...UnmarshalDefaults)
	UnmarshalKey(key string, val interface{}, additionalDefaults ...UnmarshalDefaults)
}

//go:generate mockery -name GosoConf
type GosoConf interface {
	Config
	Option(options ...Option) error
}

type config struct {
	lookupEnv      LookupEnv
	errorHandlers  []ErrorHandler
	sanitizers     []Sanitizer
	settings       *Map
	envKeyPrefix   string
	envKeyReplacer *strings.Replacer
}

var templateRegex = regexp.MustCompile("{([\\w.\\-]+)}")

func New() GosoConf {
	return NewWithInterfaces(os.LookupEnv)
}

func NewWithInterfaces(lookupEnv LookupEnv) GosoConf {
	cfg := &config{
		lookupEnv:     lookupEnv,
		errorHandlers: []ErrorHandler{defaultErrorHandler},
		sanitizers:    make([]Sanitizer, 0),
		settings:      NewMap(),
	}

	return cfg
}

func (c *config) AllKeys() []string {
	return funk.Keys(c.settings.Msi()).([]string)
}

func (c *config) AllSettings() map[string]interface{} {
	return c.settings.Msi()
}

func (c *config) Get(key string, optionalDefault ...interface{}) interface{} {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	return c.get(key)
}

func (c *config) GetBool(key string, optionalDefault ...bool) bool {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	data := c.get(key)
	b, err := cast.ToBoolE(data)

	if err != nil {
		c.err(err, "can not cast value %v[%T] of key %s to bool", data, data, key)
		return false
	}

	return b
}

func (c *config) GetDuration(key string, optionalDefault ...time.Duration) time.Duration {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	data := c.get(key)
	duration, err := cast.ToDurationE(data)

	if err != nil {
		c.err(err, "can not cast value %v[%T] of key %s to duration", data, data, key)
		return time.Duration(0)
	}

	return duration
}

func (c *config) GetInt(key string, optionalDefault ...int) int {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	data := c.get(key)
	i, err := cast.ToIntE(data)

	if err != nil {
		c.err(err, "can not cast value %v[%T] of key %s to int", data, data, key)
		return 0
	}

	return i
}

func (c *config) GetIntSlice(key string, optionalDefault ...[]int) []int {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	data := c.get(key)
	intSlice, err := cast.ToIntSliceE(data)

	if err != nil {
		c.err(err, "can not cast value %v[%T] of key %s to []int", data, data, key)
		return nil
	}

	return intSlice
}

func (c *config) GetFloat64(key string, optionalDefault ...float64) float64 {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	data := c.get(key)
	i, err := cast.ToFloat64E(data)

	if err != nil {
		c.err(err, "can not cast value %v[%T] of key %s to float64", data, data, key)
		return 0.0
	}

	return i
}

func (c *config) GetMsiSlice(key string, optionalDefault ...[]map[string]interface{}) []map[string]interface{} {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	var err error
	var data = c.settings.Get(key)
	var reflectValue = reflect.ValueOf(data)

	if reflectValue.Kind() != reflect.Slice {
		err = fmt.Errorf("can not cast value %v[%T] of key %s to []map[string]interface{}", data, data, key)
		c.err(err, err.Error())
		return nil
	}

	var ok bool
	var element interface{}
	var msi map[string]interface{}
	var msiSlice = make([]map[string]interface{}, reflectValue.Len())

	for i := 0; i < reflectValue.Len(); i++ {
		element = reflectValue.Index(i).Interface()

		if msi, ok = element.(map[string]interface{}); !ok {
			err := fmt.Errorf("element of key %s should be a msi but instead is %T", key, element)
			c.err(err, err.Error())
			return nil
		}

		msiSlice[i] = msi
	}

	return msiSlice
}

func (c *config) GetString(key string, optionalDefault ...string) string {
	return c.getString(key, optionalDefault...)
}

func (c *config) GetStringMap(key string, optionalDefault ...map[string]interface{}) map[string]interface{} {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	data := c.get(key)
	strMap, err := cast.ToStringMapE(data)

	if err != nil {
		c.err(err, "can not cast value %v[%T] of key %s to map[string]interface{}", data, data, key)
		return nil
	}

	for k, v := range strMap {
		if str, ok := v.(string); ok {
			strMap[k] = c.augmentString(str)
		}
	}

	return strMap
}

func (c *config) GetStringMapString(key string, optionalDefault ...map[string]string) map[string]string {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	data := c.get(key)
	strMap, err := cast.ToStringMapStringE(data)

	if err != nil {
		c.err(err, "can not cast value %v[%T] of key %s to map[string]string", data, data, key)
		return nil
	}

	for k, v := range strMap {
		strMap[k] = c.augmentString(v)
	}

	return strMap
}

func (c *config) GetStringSlice(key string, optionalDefault ...[]string) []string {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	var err error
	var strSlice []string

	data := c.get(key)

	switch d := data.(type) {
	case string:
		strSlice = strings.Split(d, ",")
	default:
		strSlice, err = cast.ToStringSliceE(data)
	}

	if err != nil {
		c.err(err, "can not cast value %v[%T] of key %s to []string", data, data, key)
		return nil
	}

	for i := 0; i < len(strSlice); i++ {
		strSlice[i] = c.augmentString(strSlice[i])
		strSlice[i] = strings.TrimSpace(strSlice[i])
	}

	return strSlice
}

func (c *config) GetTime(key string, optionalDefault ...time.Time) time.Time {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	data := c.get(key)
	tm, err := cast.ToTimeE(data)

	if err != nil {
		c.err(err, "can not cast value %v[%T] of key %s to time.Time", data, data, key)
		return time.Time{}
	}

	return tm
}

func (c *config) IsSet(key string) bool {
	return c.isSet(key)
}

func (c *config) Option(options ...Option) error {
	for _, opt := range options {
		if err := opt(c); err != nil {
			return err
		}
	}

	return nil
}

func (c *config) UnmarshalDefaults(output interface{}, additionalDefaults ...UnmarshalDefaults) {
	refl.InitializeMapsAndSlices(output)
	finalSettings := make(map[string]interface{})

	ms := c.buildMapStruct(output)
	zeroSettings, defaults, err := ms.ReadZeroAndDefaultValues()

	if err != nil {
		c.err(err, "can not read zeros and defaults for struct %T", output)
	}

	if err := mergo.Merge(&finalSettings, zeroSettings.Value().MSI(), mergo.WithOverride); err != nil {
		c.err(err, "can not merge zero settings for struct: %T", output)
		return
	}

	if err := mergo.Merge(&finalSettings, defaults.Value().MSI(), mergo.WithOverride); err != nil {
		c.err(err, "can not merge default settings for struct: %T", output)
		return
	}

	for _, def := range additionalDefaults {
		if err := def(c, &finalSettings); err != nil {
			c.err(err, "can not merge additional default settings for struct: %T", output)
			return
		}
	}

	if err = ms.Write(finalSettings); err != nil {
		c.err(err, "can not write defaults into struct %T", output)
		return
	}
}

func (c *config) UnmarshalKey(key string, output interface{}, defaults ...UnmarshalDefaults) {
	if refl.IsPointerToStruct(output) {
		c.unmarshalStruct(key, output, defaults)
		return
	}

	if refl.IsPointerToSlice(output) {
		c.unmarshalSlice(key, output, defaults)
		return
	}

	if refl.IsPointerToMap(output) {
		c.unmarshalMap(key, output, defaults)
		return
	}

	err := fmt.Errorf("output should be a pointer to struct or slice but instead is %T", output)
	c.err(err, "can not unmarshal key %s", key)
}

func (c *config) augmentString(str string) string {
	matches := templateRegex.FindAllStringSubmatch(str, -1)

	for _, m := range matches {
		replace := fmt.Sprint(c.getString(m[1]))
		str = strings.Replace(str, m[0], replace, -1)
	}

	return str
}

func (c *config) err(err error, msg string, args ...interface{}) {
	for i := 0; i < len(c.errorHandlers); i++ {
		c.errorHandlers[i](err, msg, args...)
	}
}

func (c *config) buildMapStruct(target interface{}) *MapStruct {
	ms, err := NewMapStruct(target, &MapStructSettings{
		FieldTag:   "cfg",
		DefaultTag: "default",
		Casters: []MapStructCaster{
			MapStructDurationCaster,
			MapStructSliceCaster,
			MapStructTimeCaster,
		},
		Decoders: []MapStructDecoder{
			c.decodeAugmentHook(),
		},
	})

	if err != nil {
		c.err(err, "can not create MapStruct for target %T", target)
		return nil
	}

	return ms
}

func (c *config) decodeAugmentHook() MapStructDecoder {
	return func(_ reflect.Type, val interface{}) (interface{}, error) {
		if raw, ok := val.(string); ok {
			return c.augmentString(raw), nil
		}

		return val, nil
	}
}

func (c *config) get(key string) interface{} {
	value := map[string]interface{}{
		key: c.settings.Get(key),
	}

	environment := c.readEnvironment(c.envKeyPrefix, value)

	if err := mergo.Merge(&value, environment, mergo.WithOverride); err != nil {
		c.err(err, "can not merge environment into result")
		return nil
	}

	c.settings.Set(key, value[key])

	return value[key]
}

func (c *config) getString(key string, optionalDefault ...string) string {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return c.augmentString(optionalDefault[0])
	}

	data := c.get(key)
	str, err := cast.ToStringE(data)

	if err != nil {
		panic(fmt.Errorf("can not cast value %v of key %s to string", data, key))
	}

	augmented := c.augmentString(str)

	return augmented
}

func (c *config) isSet(key string) bool {
	envKey := c.resolveEnvKey(c.envKeyPrefix, key)
	if _, ok := c.lookupEnv(envKey); ok {
		return true
	}

	return c.settings.Has(key)
}

func (c *config) keyCheck(key string, defaults int) bool {
	if c.isSet(key) {
		return true
	}

	if defaults > 0 {
		return false
	}

	err := fmt.Errorf("there is no config setting for key '%v'", key)
	c.err(err, "key check failed")

	return false
}

func (c *config) merge(prefix string, setting interface{}, options ...MapOption) error {
	if msi, ok := setting.(map[string]interface{}); ok {
		return c.mergeMsi(prefix, msi, options...)
	}

	if refl.IsStructOrPointerToStruct(setting) {
		return c.mergeStruct(prefix, setting, options...)
	}

	return c.mergeValue(prefix, setting, options...)
}

func (c *config) mergeValue(prefix string, value interface{}, options ...MapOption) error {
	sanitizedValue, err := Sanitize("root", value, c.sanitizers)

	if err != nil {
		return fmt.Errorf("could not sanitize settings on merge: %w", err)
	}

	c.settings.Set(prefix, sanitizedValue, options...)

	return nil
}

func (c *config) mergeMsi(prefix string, settings map[string]interface{}, options ...MapOption) error {
	sanitizedSettings, err := Sanitize("root", settings, c.sanitizers)

	if err != nil {
		return fmt.Errorf("could not sanitize settings on merge: %w", err)
	}

	c.settings.Merge(prefix, sanitizedSettings, options...)

	return nil
}

func (c *config) mergeStruct(prefix string, settings interface{}, options ...MapOption) error {
	ms := c.buildMapStruct(settings)
	msi, err := ms.Read()

	if err != nil {
		return err
	}

	return c.mergeMsi(prefix, msi.Msi(), options...)
}

func (c *config) readEnvironment(prefix string, input map[string]interface{}) map[string]interface{} {
	environment := make(map[string]interface{})

	for k, v := range input {
		key := c.resolveEnvKey(prefix, k)

		if nested, ok := v.(map[string]interface{}); ok {
			environment[k] = c.readEnvironment(key, nested)
			continue
		}

		if envValue, ok := c.lookupEnv(key); ok {
			environment[k] = c.augmentString(envValue)
		}
	}

	return environment
}

func (c *config) resolveEnvKey(prefix string, key string) string {
	if len(prefix) > 0 {
		key = strings.Join([]string{prefix, key}, ".")
	}

	rp := regexp.MustCompile("\\[(\\d)\\]")
	matches := rp.FindAllStringSubmatch(key, -1)

	for _, m := range matches {
		key = strings.Replace(key, m[0], fmt.Sprintf(".%s", m[1]), -1)
	}

	if c.envKeyReplacer != nil {
		key = c.envKeyReplacer.Replace(key)
	}

	return strings.ToUpper(key)
}

func (c *config) unmarshalMap(key string, output interface{}, defaults []UnmarshalDefaults) {
	names := c.GetStringMap(key)
	m, err := refl.MapOf(output)

	if err != nil {
		c.err(err, "can not unmarshal key %s", key)
		return
	}

	for name := range names {
		keyIndex := fmt.Sprintf("%s.%s", key, name)
		item := m.NewElement()

		c.unmarshalStruct(keyIndex, item, defaults)
		err = m.Set(name, item)

		if err != nil {
			c.err(err, "can not unmarshal key %s", key)
			return
		}
	}
}

func (c *config) unmarshalSlice(key string, output interface{}, defaults []UnmarshalDefaults) {
	data := c.settings.Get(key)
	rv := reflect.ValueOf(data)

	if rv.Kind() != reflect.Slice {
		err := fmt.Errorf("data for key %s should be of type []interface{} but instead is of type %T", key, data)
		c.err(err, "can not unmarshal key %s", key)
		return
	}

	slice, err := refl.SliceOf(output)

	if err != nil {
		c.err(err, "can not unmarshal key %s into slice", key)
		return
	}

	for i := 0; i < rv.Len(); i++ {
		keyIndex := fmt.Sprintf("%s[%d]", key, i)
		elem := slice.NewElement()

		c.unmarshalStruct(keyIndex, elem, defaults)

		if err := slice.Append(elem); err != nil {
			c.err(err, "can not unmarshal key %s into slice", key)
			return
		}
	}
}

func (c *config) unmarshalStruct(key string, output interface{}, additionalDefaults []UnmarshalDefaults) {
	refl.InitializeMapsAndSlices(output)
	finalSettings := make(map[string]interface{})

	ms := c.buildMapStruct(output)
	zeroSettings, defaults, err := ms.ReadZeroAndDefaultValues()

	if err != nil {
		c.err(err, "can not read zeros and defaults for key %s", key)
	}

	if err := mergo.Merge(&finalSettings, zeroSettings.Value().MSI(), mergo.WithOverride); err != nil {
		c.err(err, "can not merge zero settings for key: %s", key)
		return
	}

	if err := mergo.Merge(&finalSettings, defaults.Value().MSI(), mergo.WithOverride); err != nil {
		c.err(err, "can not merge default settings for key: %s", key)
		return
	}

	for _, def := range additionalDefaults {
		if err := def(c, &finalSettings); err != nil {
			c.err(err, "can not merge additional default settings for key: %s", key)
			return
		}
	}

	if c.settings.Has(key) {
		data := c.settings.Get(key)

		settings, ok := data.(map[string]interface{})

		if !ok {
			c.err(errors.New("value is not of type map[string]interface{}"), "can not get settings for key: %s", key)
			return
		}

		if err := mergo.Merge(&finalSettings, settings, mergo.WithOverride); err != nil {
			c.err(err, "can not merge settings for key: %s", key)
			return
		}
	}

	environmentKey := c.resolveEnvKey(c.envKeyPrefix, key)
	environmentSettings := c.readEnvironment(environmentKey, finalSettings)

	if err := mergo.Merge(&finalSettings, environmentSettings, mergo.WithOverride); err != nil {
		c.err(err, "can not merge zero settings for key: %s", key)
		return
	}

	c.settings.Set(key, finalSettings)

	if err = ms.Write(finalSettings); err != nil {
		c.err(err, "error unmarshalling key: %s", key)
		return
	}

	validate := validator.New()
	err = validate.Struct(output)

	if err == nil {
		return
	}

	if _, ok := err.(*validator.InvalidValidationError); ok {
		c.err(err, "can not validate result of key: %s", key)
		return
	}

	errs := &multierror.Error{}
	for _, validationErr := range err.(validator.ValidationErrors) {
		err := fmt.Errorf("the setting %s with value %v does not match its requirement", validationErr.Field(), validationErr.Value())
		errs = multierror.Append(errs, err)
	}

	if errs != nil {
		c.err(errs, "validation failed for key: %s", key)
		return
	}
}
