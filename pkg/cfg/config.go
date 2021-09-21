package cfg

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/spf13/cast"
	"github.com/thoas/go-funk"
)

type LookupEnv func(key string) (string, bool)

//go:generate mockery --name Config
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

//go:generate mockery --name GosoConf
type GosoConf interface {
	Config
	Option(options ...Option) error
}

type config struct {
	envProvider    EnvProvider
	errorHandlers  []ErrorHandler
	sanitizers     []Sanitizer
	settings       *mapx.MapX
	envKeyPrefix   string
	envKeyReplacer *strings.Replacer
}

var (
	DefaultEnvKeyReplacer = strings.NewReplacer(".", "_", "-", "_")
	templateRegexp        = regexp.MustCompile(`{([\w.\-]+)}`)
	keyToEnvRegexp        = regexp.MustCompile(`\[(\d+)\]`)
)

func New() GosoConf {
	return NewWithInterfaces(NewOsEnvProvider())
}

func NewWithInterfaces(envProvider EnvProvider) GosoConf {
	cfg := &config{
		envProvider:   envProvider,
		errorHandlers: []ErrorHandler{defaultErrorHandler},
		sanitizers:    make([]Sanitizer, 0),
		settings:      mapx.NewMapX(),
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
		c.err("can not cast value %v[%T] of key %s to bool: %w", data, data, key, err)
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
		c.err("can not cast value %v[%T] of key %s to duration: %w", data, data, key, err)
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
		c.err("can not cast value %v[%T] of key %s to int: %w", data, data, key, err)
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
		c.err("can not cast value %v[%T] of key %s to []int: %w", data, data, key, err)
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
		c.err("can not cast value %v[%T] of key %s to float64: %w", data, data, key, err)
		return 0.0
	}

	return i
}

func (c *config) GetMsiSlice(key string, optionalDefault ...[]map[string]interface{}) []map[string]interface{} {
	if ok := c.keyCheck(key, len(optionalDefault)); !ok && len(optionalDefault) > 0 {
		return optionalDefault[0]
	}

	var err error
	data := c.settings.Get(key).Data()
	reflectValue := reflect.ValueOf(data)

	if reflectValue.Kind() != reflect.Slice {
		c.err("can not cast value %v[%T] of key %s to []map[string]interface{}: %w", data, data, key, err)
		return nil
	}

	var ok bool
	var element interface{}
	var msi map[string]interface{}
	msiSlice := make([]map[string]interface{}, reflectValue.Len())

	for i := 0; i < reflectValue.Len(); i++ {
		element = reflectValue.Index(i).Interface()

		if msi, ok = element.(map[string]interface{}); !ok {
			c.err("element of key %s should be a msi but instead is %T", key, element)
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
		c.err("can not cast value %v[%T] of key %s to map[string]interface{}: %w", data, data, key, err)
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
		c.err("can not cast value %v[%T] of key %s to map[string]string: %w", data, data, key, err)
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
		c.err("can not cast value %v[%T] of key %s to []string: %w", data, data, key, err)
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
		c.err("can not cast value %v[%T] of key %s to time.Time: %w", data, data, key, err)
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
	finalSettings := mapx.NewMapX()

	ms := c.buildMapStruct(output)
	zeroSettings, defaults, err := ms.ReadZeroAndDefaultValues()
	if err != nil {
		c.err("can not read zeros and defaults for struct %T: %w", output, err)
	}

	finalSettings.Merge(".", zeroSettings)
	finalSettings.Merge(".", defaults)

	for _, def := range additionalDefaults {
		def(c, finalSettings)
	}

	if err = ms.Write(finalSettings); err != nil {
		c.err("can not write defaults into struct %T: %w", output, err)
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
	c.err("can not unmarshal key %s: %w", key, err)
}

func (c *config) augmentString(str string) string {
	matches := templateRegexp.FindAllStringSubmatch(str, -1)

	for _, m := range matches {
		replace := fmt.Sprint(c.getString(m[1]))
		str = strings.Replace(str, m[0], replace, -1)
	}

	return str
}

func (c *config) err(msg string, args ...interface{}) {
	for i := 0; i < len(c.errorHandlers); i++ {
		c.errorHandlers[i](msg, args...)
	}
}

func (c *config) buildMapStruct(target interface{}) *mapx.Struct {
	ms, err := mapx.NewStruct(target, &mapx.StructSettings{
		FieldTag:   "cfg",
		DefaultTag: "default",
		Casters: []mapx.MapStructCaster{
			mapx.MapStructDurationCaster,
			mapx.MapStructTimeCaster,
		},
		Decoders: []mapx.MapStructDecoder{
			c.decodeAugmentHook(),
		},
	})
	if err != nil {
		c.err("can not create MapXStruct for target %T: %w", target, err)
		return nil
	}

	return ms
}

func (c *config) decodeAugmentHook() mapx.MapStructDecoder {
	return func(_ reflect.Type, val interface{}) (interface{}, error) {
		if raw, ok := val.(string); ok {
			return c.augmentString(raw), nil
		}

		return val, nil
	}
}

func (c *config) get(key string) interface{} {
	data := c.settings.Get(key).Data()

	dataMap := mapx.NewMapX()
	dataMap.Set(key, data)

	environment := c.readEnvironmentFromValues(c.envKeyPrefix, dataMap)
	dataMap.Merge(".", environment)

	c.settings.Merge(".", dataMap)

	return dataMap.Get(key).Data()
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
	if _, ok := c.envProvider.LookupEnv(envKey); ok {
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
	c.err("key check failed: %w", err)

	return false
}

func (c *config) merge(prefix string, setting interface{}, options ...MergeOption) error {
	if msi, ok := setting.(map[string]interface{}); ok {
		return c.mergeMsi(prefix, msi, options...)
	}

	if refl.IsStructOrPointerToStruct(setting) {
		return c.mergeStruct(prefix, setting, options...)
	}

	return c.mergeValue(prefix, setting, options...)
}

func (c *config) mergeValue(prefix string, value interface{}, options ...MergeOption) error {
	sanitizedValue, err := Sanitize("root", value, c.sanitizers)
	if err != nil {
		return fmt.Errorf("could not sanitize settings on merge: %w", err)
	}

	mapOptions := mergeToMapOptions(options)
	c.settings.Set(prefix, sanitizedValue, mapOptions...)

	return nil
}

func (c *config) mergeMsi(prefix string, settings map[string]interface{}, options ...MergeOption) error {
	sanitizedSettings, err := Sanitize("root", settings, c.sanitizers)
	if err != nil {
		return fmt.Errorf("could not sanitize settings on merge: %w", err)
	}

	mapOptions := mergeToMapOptions(options)
	c.settings.Merge(prefix, sanitizedSettings, mapOptions...)

	return nil
}

func (c *config) mergeStruct(prefix string, settings interface{}, options ...MergeOption) error {
	ms := c.buildMapStruct(settings)
	nodeMap, err := ms.Read()
	if err != nil {
		return err
	}

	msi := nodeMap.Msi()

	return c.mergeMsi(prefix, msi, options...)
}

func (c *config) readEnvironmentFromStructKeys(prefix string, structKeys []mapx.StructKey) *mapx.MapX {
	environment := mapx.NewMapX()

	for _, structKey := range structKeys {
		switch structKey.Kind {
		case reflect.Slice:
			if len(structKey.SubKeys) > 0 {
				for i := 0; ; i++ {
					sliceKeyIndexed := fmt.Sprintf("%s[%d]", structKey.Key, i)
					sliceKeyPrefixed := fmt.Sprintf("%s.%s", prefix, sliceKeyIndexed)
					sliceValues := c.readEnvironmentFromStructKeys(sliceKeyPrefixed, structKey.SubKeys)

					if len(sliceValues.Msi()) == 0 {
						break
					}

					environment.Set(sliceKeyIndexed, sliceValues)
				}
			} else {
				for i := 0; ; i++ {
					sliceKeyIndexed := fmt.Sprintf("%s[%d]", structKey.Key, i)
					envKey := c.resolveEnvKey(prefix, sliceKeyIndexed)

					if envValue, ok := c.envProvider.LookupEnv(envKey); ok {
						augmentedString := c.augmentString(envValue)
						environment.Set(sliceKeyIndexed, augmentedString)
					} else {
						break
					}
				}
			}

		default:
			key := structKey.Key
			envKey := c.resolveEnvKey(prefix, key)

			if envValue, ok := c.envProvider.LookupEnv(envKey); ok {
				augmentedString := c.augmentString(envValue)
				environment.Set(key, augmentedString)
			}
		}
	}

	return environment
}

func (c *config) readEnvironmentFromValues(prefix string, input *mapx.MapX) *mapx.MapX {
	environment := mapx.NewMapX()

	for _, k := range input.Keys() {
		key := c.resolveEnvKey(prefix, k)
		val := input.Get(k)

		if nestedMap, err := val.Map(); err == nil {
			nestedValues := c.readEnvironmentFromValues(key, nestedMap)
			environment.Set(k, nestedValues)
			continue
		}

		if envValue, ok := c.envProvider.LookupEnv(key); ok {
			augmentedString := c.augmentString(envValue)
			environment.Set(k, augmentedString)
		}
	}

	return environment
}

func (c *config) resolveEnvKey(prefix string, key string) string {
	if len(prefix) > 0 {
		key = strings.Join([]string{prefix, key}, ".")
	}

	matches := keyToEnvRegexp.FindAllStringSubmatch(key, -1)

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
		c.err("can not unmarshal key %s: %w", key, err)
		return
	}

	for name := range names {
		keyIndex := fmt.Sprintf("%s.%s", key, name)
		item := m.NewElement()

		c.unmarshalStruct(keyIndex, item, defaults)
		err = m.Set(name, item)

		if err != nil {
			c.err("can not unmarshal key %s: %w", key, err)
			return
		}
	}
}

func (c *config) unmarshalSlice(key string, output interface{}, defaults []UnmarshalDefaults) {
	data, err := c.settings.Get(key).Slice()
	if err != nil {
		c.err("can not unmarshal key %s: %w", key, err)
		return
	}

	slice, err := refl.SliceOf(output)
	if err != nil {
		c.err("can not unmarshal key %s into slice: %w", key, err)
		return
	}

	for i := 0; i < len(data); i++ {
		keyIndex := fmt.Sprintf("%s[%d]", key, i)
		elem := slice.NewElement()

		c.unmarshalStruct(keyIndex, elem, defaults)

		if err := slice.Append(elem); err != nil {
			c.err("can not unmarshal key %s into slice: %w", key, err)
			return
		}
	}
}

func (c *config) unmarshalStruct(key string, output interface{}, additionalDefaults []UnmarshalDefaults) {
	refl.InitializeMapsAndSlices(output)
	finalSettings := mapx.NewMapX()

	ms := c.buildMapStruct(output)
	zeroSettings, defaults, err := ms.ReadZeroAndDefaultValues()
	if err != nil {
		c.err("can not read zeros and defaults for key %s: %w", key, err)
	}

	finalSettings.Merge(".", zeroSettings)
	finalSettings.Merge(".", defaults)

	for _, def := range additionalDefaults {
		def(c, finalSettings)
	}

	if c.settings.Has(key) {
		settings, err := c.settings.Get(key).Map()
		if err != nil {
			c.err("can not get settings for key: %s: %w", key, err)
			return
		}

		finalSettings.Merge(".", settings)
	}

	environmentKey := c.resolveEnvKey(c.envKeyPrefix, key)
	environmentKeySettings := c.readEnvironmentFromStructKeys(environmentKey, ms.Keys())
	environmentValueSettings := c.readEnvironmentFromValues(environmentKey, finalSettings)

	finalSettings.Merge(".", environmentKeySettings)
	finalSettings.Merge(".", environmentValueSettings)

	c.settings.Set(key, finalSettings)

	if err = ms.Write(finalSettings); err != nil {
		c.err("error unmarshalling key: %s: %w", key, err)
		return
	}

	validate := validator.New()
	err = validate.Struct(output)

	if err == nil {
		return
	}

	if _, ok := err.(*validator.InvalidValidationError); ok {
		c.err("can not validate result of key: %s: %w", key, err)
		return
	}

	errs := &multierror.Error{}
	for _, validationErr := range err.(validator.ValidationErrors) {
		err := fmt.Errorf("the setting %s with value %v does not match its requirement", validationErr.Field(), validationErr.Value())
		errs = multierror.Append(errs, err)
	}

	if errs != nil {
		c.err("validation failed for key: %s: %w", key, errs)
		return
	}
}
