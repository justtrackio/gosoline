package cfg

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hashicorp/go-multierror"
	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/mapx"
	"github.com/justtrackio/gosoline/pkg/refl"
	"github.com/spf13/cast"
)

const (
	flagNoDecode = "nodecode"
)

type LookupEnv func(key string) (string, bool)

//go:generate go run github.com/vektra/mockery/v2 --name Config
type Config interface {
	AllKeys() []string
	AllSettings() map[string]any
	Get(key string, optionalDefault ...any) (any, error)
	GetBool(key string, optionalDefault ...bool) (bool, error)
	GetDuration(key string, optionalDefault ...time.Duration) (time.Duration, error)
	GetInt(key string, optionalDefault ...int) (int, error)
	GetIntSlice(key string, optionalDefault ...[]int) ([]int, error)
	GetFloat64(key string, optionalDefault ...float64) (float64, error)
	GetMsiSlice(key string, optionalDefault ...[]map[string]any) ([]map[string]any, error)
	GetString(key string, optionalDefault ...string) (string, error)
	GetStringMap(key string, optionalDefault ...map[string]any) (map[string]any, error)
	GetStringMapString(key string, optionalDefault ...map[string]string) (map[string]string, error)
	GetStringSlice(key string, optionalDefault ...[]string) ([]string, error)
	GetTime(key string, optionalDefault ...time.Time) (time.Time, error)
	FormatString(pattern string, args ...map[string]string) (string, error)
	IsSet(string) bool
	HasPrefix(prefix string) bool
	UnmarshalDefaults(val any, additionalDefaults ...UnmarshalDefaults) error
	UnmarshalKey(key string, val any, additionalDefaults ...UnmarshalDefaults) error
}

//go:generate go run github.com/vektra/mockery/v2 --name GosoConf
type GosoConf interface {
	Config
	Option(options ...Option) error
}

type config struct {
	envProvider    EnvProvider
	sanitizers     []Sanitizer
	settings       *mapx.MapX
	envKeyPrefix   string
	envKeyReplacer *strings.Replacer
}

var (
	DefaultEnvKeyReplacer = strings.NewReplacer(".", "_", "-", "_")
	valFlagsRegexp        = regexp.MustCompile(`(!(\S*)\s)?(.*)`)
	templateRegexp        = regexp.MustCompile(`{([\w.\-]+)}`)
	keyToEnvRegexp        = regexp.MustCompile(`\[(\d+)\]`)
)

func New(msis ...map[string]any) GosoConf {
	return NewWithInterfaces(NewOsEnvProvider(), msis...)
}

func NewWithInterfaces(envProvider EnvProvider, msis ...map[string]any) GosoConf {
	cfg := &config{
		envProvider: envProvider,
		sanitizers:  make([]Sanitizer, 0),
		settings:    mapx.NewMapX(msis...),
	}

	return cfg
}

func (c *config) AllKeys() []string {
	return funk.Keys(c.settings.Msi())
}

func (c *config) AllSettings() map[string]any {
	return c.settings.Msi()
}

func (c *config) Get(key string, optionalDefault ...any) (any, error) {
	return c.get(key, optionalDefault)
}

func (c *config) GetBool(key string, optionalDefault ...bool) (b bool, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return false, err
	}

	if b, err = cast.ToBoolE(data); err != nil {
		return false, fmt.Errorf("can not cast value %v[%T] of key %s to bool: %w", data, data, key, err)
	}

	return b, nil
}

func (c *config) GetDuration(key string, optionalDefault ...time.Duration) (duration time.Duration, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return time.Duration(0), err
	}

	if duration, err = cast.ToDurationE(data); err != nil {
		return time.Duration(0), fmt.Errorf("can not cast value %v[%T] of key %s to duration: %w", data, data, key, err)
	}

	return duration, nil
}

func (c *config) GetInt(key string, optionalDefault ...int) (i int, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return 0, err
	}

	if i, err = cast.ToIntE(data); err != nil {
		return 0, fmt.Errorf("can not cast value %v[%T] of key %s to int: %w", data, data, key, err)
	}

	return i, nil
}

func (c *config) GetIntSlice(key string, optionalDefault ...[]int) (intSlice []int, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return nil, err
	}

	if intSlice, err = cast.ToIntSliceE(data); err != nil {
		return nil, fmt.Errorf("can not cast value %v[%T] of key %s to []int: %w", data, data, key, err)
	}

	return intSlice, nil
}

func (c *config) GetFloat64(key string, optionalDefault ...float64) (f float64, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return 0.0, err
	}

	if f, err = cast.ToFloat64E(data); err != nil {
		return 0.0, fmt.Errorf("can not cast value %v[%T] of key %s to float64: %w", data, data, key, err)
	}

	return f, nil
}

func (c *config) GetMsiSlice(key string, optionalDefault ...[]map[string]any) (msiSlice []map[string]any, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return nil, err
	}

	reflectValue := reflect.ValueOf(data)
	if reflectValue.Kind() != reflect.Slice {
		return nil, fmt.Errorf("can not cast value %v[%T] of key %s to []map[string]any", data, data, key)
	}

	var ok bool
	var element any
	var msi map[string]any
	msiSlice = make([]map[string]any, reflectValue.Len())

	for i := 0; i < reflectValue.Len(); i++ {
		element = reflectValue.Index(i).Interface()

		if msi, ok = element.(map[string]any); !ok {
			return nil, fmt.Errorf("element of key %s should be a msi but instead is %T", key, element)
		}

		msiSlice[i] = msi
	}

	return msiSlice, nil
}

func (c *config) GetString(key string, optionalDefault ...string) (string, error) {
	return c.getString(key, optionalDefault...)
}

func (c *config) GetStringMap(key string, optionalDefault ...map[string]any) (strMap map[string]any, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return nil, err
	}

	if strMap, err = cast.ToStringMapE(data); err != nil {
		return nil, fmt.Errorf("can not cast value %v[%T] of key %s to map[string]any: %w", data, data, key, err)
	}

	for k, v := range strMap {
		if str, ok := v.(string); ok {
			augmented, err := c.augmentString(str)
			if err != nil {
				return nil, fmt.Errorf("can not augment string in map for key %s: %w", key, err)
			}
			strMap[k] = augmented
		}
	}

	return strMap, nil
}

func (c *config) GetStringMapString(key string, optionalDefault ...map[string]string) (strMap map[string]string, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return nil, err
	}

	if strMap, err = cast.ToStringMapStringE(data); err != nil {
		return nil, fmt.Errorf("can not cast value %v[%T] of key %s to map[string]string: %w", data, data, key, err)
	}

	for k, v := range strMap {
		augmented, err := c.augmentString(v)
		if err != nil {
			return nil, fmt.Errorf("can not augment string in map for key %s: %w", key, err)
		}
		strMap[k] = augmented
	}

	return strMap, nil
}

func (c *config) GetStringSlice(key string, optionalDefault ...[]string) (strSlice []string, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return nil, err
	}

	switch d := data.(type) {
	case string:
		strSlice = strings.Split(d, ",")
	default:
		strSlice, err = cast.ToStringSliceE(data)
	}

	if err != nil {
		return nil, fmt.Errorf("can not cast value %v[%T] of key %s to []string: %w", data, data, key, err)
	}

	for i := 0; i < len(strSlice); i++ {
		augmented, err := c.augmentString(strSlice[i])
		if err != nil {
			return nil, fmt.Errorf("can not augment string in slice for key %s: %w", key, err)
		}
		strSlice[i] = strings.TrimSpace(augmented)
	}

	return strSlice, nil
}

func (c *config) GetTime(key string, optionalDefault ...time.Time) (tm time.Time, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return time.Time{}, err
	}

	if tm, err = cast.ToTimeE(data); err != nil {
		return time.Time{}, fmt.Errorf("can not cast value %v[%T] of key %s to time.Time: %w", data, data, key, err)
	}

	return tm, nil
}

func (c *config) FormatString(pattern string, args ...map[string]string) (string, error) {
	return c.augmentString(pattern, args...)
}

func (c *config) IsSet(key string) bool {
	return c.isSet(key)
}

func (c *config) HasPrefix(prefix string) bool {
	envPrefix := c.resolveEnvKey(c.envKeyPrefix, prefix)

	if c.envProvider.PrefixExists(envPrefix) {
		return true
	}

	return c.IsSet(prefix)
}

func (c *config) Option(options ...Option) error {
	for _, opt := range options {
		if err := opt(c); err != nil {
			return err
		}
	}

	return nil
}

func (c *config) UnmarshalDefaults(output any, additionalDefaults ...UnmarshalDefaults) error {
	refl.InitializeMapsAndSlices(output)
	finalSettings := mapx.NewMapX()

	var err error
	var ms *mapx.Struct
	var zeroSettings, defaults *mapx.MapX

	if ms, err = c.buildMapStruct(output); err != nil {
		return fmt.Errorf("can not build mapx.Struct for output: %w", err)
	}

	if zeroSettings, defaults, err = ms.ReadZeroAndDefaultValues(); err != nil {
		return fmt.Errorf("can not read zeros and defaults for struct %T: %w", output, err)
	}

	finalSettings.Merge(".", zeroSettings)
	finalSettings.Merge(".", defaults)

	for _, def := range additionalDefaults {
		if err := def(c, finalSettings); err != nil {
			return fmt.Errorf("can not apply additional defaults: %w", err)
		}
	}

	if err = ms.Write(finalSettings); err != nil {
		return fmt.Errorf("can not write defaults into struct %T: %w", output, err)
	}

	return nil
}

func (c *config) UnmarshalKey(key string, output any, defaults ...UnmarshalDefaults) error {
	if refl.IsPointerToStruct(output) {
		if err := c.unmarshalStruct(key, output, defaults); err != nil {
			return fmt.Errorf("can not unmarshal config struct with key %s: %w", key, err)
		}

		return nil
	}

	if refl.IsPointerToSlice(output) {
		if err := c.unmarshalSlice(key, output, defaults); err != nil {
			return fmt.Errorf("can not unmarshal config struct with key %s: %w", key, err)
		}

		return nil
	}

	if refl.IsPointerToMap(output) {
		if err := c.unmarshalMap(key, output, defaults); err != nil {
			return fmt.Errorf("can not unmarshal config struct with key %s: %w", key, err)
		}

		return nil
	}

	return fmt.Errorf("can not unmarshal key %s: output should be a pointer to struct or slice but instead is %T", key, output)
}

func (c *config) augmentString(str string, args ...map[string]string) (string, error) {
	groups := valFlagsRegexp.FindStringSubmatch(str)
	flags := make([]string, 0)

	if groups[2] != "" {
		flags = strings.Split(groups[2], ",")
		flags = funk.Map(flags, strings.ToLower)
		str = groups[3]
	}

	if funk.Contains(flags, flagNoDecode) {
		return str, nil
	}

	matches := templateRegexp.FindAllStringSubmatch(str, -1)
	allArgs := funk.MergeMaps(args...)

	var ok bool
	var err error
	var replace string

	for _, m := range matches {
		if replace, ok = allArgs[m[1]]; ok {
			str = strings.ReplaceAll(str, m[0], replace)

			continue
		}

		if replace, err = c.getString(m[1]); err != nil {
			return "", err
		}

		str = strings.ReplaceAll(str, m[0], replace)
	}

	return str, nil
}

func (c *config) buildMapStruct(target any) (*mapx.Struct, error) {
	ms, err := mapx.NewStruct(target, &mapx.StructSettings{
		FieldTag:   "cfg",
		DefaultTag: "default",
		Casters: []mapx.MapStructCaster{
			mapx.MapStructDurationCaster,
			mapx.MapStructSliceCaster,
			mapx.MapStructTimeCaster,
		},
		Decoders: []mapx.MapStructDecoder{
			c.decodeAugmentHook(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("can not create MapXStruct for target %T: %w", target, err)
	}

	return ms, nil
}

func (c *config) decodeAugmentHook() mapx.MapStructDecoder {
	return func(_ reflect.Type, val any) (any, error) {
		if raw, ok := val.(string); ok {
			return c.augmentString(raw)
		}

		return val, nil
	}
}

func (c *config) get(key string, optionalDefault ...any) (any, error) {
	var err error
	var defaults []any

	if len(optionalDefault) > 0 {
		if defaults, err = refl.InterfaceToInterfaceSlice(optionalDefault[0]); err != nil {
			return nil, fmt.Errorf("can not convert optional default value %v[%T] to []any: %w", optionalDefault[0], optionalDefault[0], err)
		}
	}

	if !c.isSet(key) && len(defaults) == 0 {
		return nil, fmt.Errorf("there is no config setting or default for key %q", key)
	}

	if !c.isSet(key) && len(defaults) > 0 {
		return defaults[0], nil
	}

	data := c.settings.Get(key).Data()

	dataMap := mapx.NewMapX()
	dataMap.Set(key, data)

	environment, err := c.readEnvironmentFromValues(c.envKeyPrefix, dataMap)
	if err != nil {
		return nil, fmt.Errorf("could not read environment from values: %w", err)
	}

	dataMap.Merge(".", environment)
	c.settings.Merge(".", dataMap)

	return dataMap.Get(key).Data(), nil
}

func (c *config) getString(key string, optionalDefault ...string) (str string, err error) {
	var data any
	if data, err = c.get(key, optionalDefault); err != nil {
		return "", err
	}

	if str, err = cast.ToStringE(data); err != nil {
		return "", fmt.Errorf("can not cast value %v[%T] of key %s to string: %w", data, data, key, err)
	}

	return c.augmentString(str)
}

func (c *config) isSet(key string) bool {
	envKey := c.resolveEnvKey(c.envKeyPrefix, key)
	if _, ok := c.envProvider.LookupEnv(envKey); ok {
		return true
	}

	return c.settings.Has(key)
}

func (c *config) merge(prefix string, setting any, options ...MergeOption) error {
	if msi, ok := setting.(map[string]any); ok {
		return c.mergeMsi(prefix, msi, options...)
	}

	if refl.IsStructOrPointerToStruct(setting) {
		return c.mergeStruct(prefix, setting, options...)
	}

	return c.mergeValue(prefix, setting, options...)
}

func (c *config) mergeValue(prefix string, value any, options ...MergeOption) error {
	sanitizedValue, err := Sanitize("root", value, c.sanitizers)
	if err != nil {
		return fmt.Errorf("could not sanitize settings on merge: %w", err)
	}

	mapOptions := mergeToMapOptions(options)
	c.settings.Set(prefix, sanitizedValue, mapOptions...)

	return nil
}

func (c *config) mergeMsi(prefix string, settings map[string]any, options ...MergeOption) error {
	sanitizedSettings, err := Sanitize("root", settings, c.sanitizers)
	if err != nil {
		return fmt.Errorf("could not sanitize settings on merge: %w", err)
	}

	mapOptions := mergeToMapOptions(options)
	c.settings.Merge(prefix, sanitizedSettings, mapOptions...)

	return nil
}

func (c *config) mergeStruct(prefix string, settings any, options ...MergeOption) error {
	var err error
	var ms *mapx.Struct
	var nodeMap *mapx.MapX

	if ms, err = c.buildMapStruct(settings); err != nil {
		return fmt.Errorf("can not build mapx.Struct for settings: %w", err)
	}

	if nodeMap, err = ms.Read(); err != nil {
		return fmt.Errorf("can not perform read on mapx.Struct: %w", err)
	}

	msi := nodeMap.Msi()

	return c.mergeMsi(prefix, msi, options...)
}

func (c *config) readEnvironmentFromStructKeys(prefix string, structKeys []mapx.StructKey) (*mapx.MapX, error) {
	environment := mapx.NewMapX()

	for _, structKey := range structKeys {
		switch structKey.Kind {
		case reflect.Slice:
			if err := c.readEnvironmentFromStructKeysSlice(prefix, structKey, environment); err != nil {
				return nil, err
			}

		default:
			key := structKey.Key
			envKey := c.resolveEnvKey(prefix, key)

			if envValue, ok := c.envProvider.LookupEnv(envKey); ok {
				augmentedString, err := c.augmentString(envValue)
				if err != nil {
					return nil, err
				}
				environment.Set(key, augmentedString)
			}
		}
	}

	return environment, nil
}

func (c *config) readEnvironmentFromStructKeysSlice(prefix string, structKey mapx.StructKey, environment *mapx.MapX) error {
	if len(structKey.SubKeys) > 0 {
		return c.readEnvironmentFromStructKeysSlicePrefixed(prefix, structKey, environment)
	}

	return c.readEnvironmentFromStructKeysSliceIndexed(prefix, structKey, environment)
}

func (c *config) readEnvironmentFromStructKeysSlicePrefixed(prefix string, structKey mapx.StructKey, environment *mapx.MapX) error {
	for i := 0; ; i++ {
		sliceKeyIndexed := fmt.Sprintf("%s[%d]", structKey.Key, i)
		sliceKeyPrefixed := fmt.Sprintf("%s.%s", prefix, sliceKeyIndexed)
		sliceValues, err := c.readEnvironmentFromStructKeys(sliceKeyPrefixed, structKey.SubKeys)
		if err != nil {
			return err
		}

		if len(sliceValues.Msi()) == 0 {
			break
		}

		environment.Set(sliceKeyIndexed, sliceValues)
	}

	return nil
}

func (c *config) readEnvironmentFromStructKeysSliceIndexed(prefix string, structKey mapx.StructKey, environment *mapx.MapX) error {
	for i := 0; ; i++ {
		sliceKeyIndexed := fmt.Sprintf("%s[%d]", structKey.Key, i)
		envKey := c.resolveEnvKey(prefix, sliceKeyIndexed)

		if envValue, ok := c.envProvider.LookupEnv(envKey); ok {
			augmentedString, err := c.augmentString(envValue)
			if err != nil {
				return err
			}
			environment.Set(sliceKeyIndexed, augmentedString)
		} else {
			break
		}
	}

	return nil
}

func (c *config) readEnvironmentFromValues(prefix string, input *mapx.MapX) (*mapx.MapX, error) {
	environment := mapx.NewMapX()

	for _, k := range input.Keys() {
		key := c.resolveEnvKey(prefix, k)
		val := input.Get(k)

		if nestedMap, err := val.Map(); err == nil {
			nestedValues, err := c.readEnvironmentFromValues(key, nestedMap)
			if err != nil {
				return nil, err
			}
			environment.Set(k, nestedValues)

			continue
		}

		if envValue, ok := c.envProvider.LookupEnv(key); ok {
			augmentedString, err := c.augmentString(envValue)
			if err != nil {
				return nil, err
			}
			environment.Set(k, augmentedString)
		}
	}

	return environment, nil
}

func (c *config) resolveEnvKey(prefix string, key string) string {
	if prefix != "" {
		key = fmt.Sprintf("%s.%s", prefix, key)
	}

	matches := keyToEnvRegexp.FindAllStringSubmatch(key, -1)

	for _, m := range matches {
		key = strings.ReplaceAll(key, m[0], fmt.Sprintf(".%s", m[1]))
	}

	if c.envKeyReplacer != nil {
		key = c.envKeyReplacer.Replace(key)
	}

	return strings.ToUpper(key)
}

func (c *config) unmarshalMap(key string, output any, defaults []UnmarshalDefaults) error {
	names, err := c.GetStringMap(key)
	if err != nil {
		return fmt.Errorf("can not get string map for key %s: %w", key, err)
	}

	m, err := refl.MapOf(output)
	if err != nil {
		return fmt.Errorf("can not unmarshal key %s: %w", key, err)
	}

	for name := range names {
		keyIndex := fmt.Sprintf("%s.%s", key, name)
		item := m.NewElement()

		cErr := c.unmarshalStruct(keyIndex, item, defaults)
		if cErr != nil {
			return fmt.Errorf("can not unmarshal key %s: %w", keyIndex, cErr)
		}

		if err = m.Set(name, item); err != nil {
			return fmt.Errorf("can not unmarshal key %s: %w", key, err)
		}
	}

	return nil
}

func (c *config) unmarshalSlice(key string, output any, defaults []UnmarshalDefaults) error {
	data, err := c.settings.Get(key).Slice()
	if err != nil {
		return fmt.Errorf("can not unmarshal key %s: %w", key, err)
	}

	slice, err := refl.SliceOf(output)
	if err != nil {
		return fmt.Errorf("can not unmarshal key %s into slice: %w", key, err)
	}

	for i := 0; i < len(data); i++ {
		keyIndex := fmt.Sprintf("%s[%d]", key, i)
		elem := slice.NewElement()

		if err = c.unmarshalStruct(keyIndex, elem, defaults); err != nil {
			return fmt.Errorf("can not unmarshal struct with index %s: %w", keyIndex, err)
		}

		if err := slice.Append(elem); err != nil {
			return fmt.Errorf("can not unmarshal key %s into slice: %w", key, err)
		}
	}

	return nil
}

func (c *config) unmarshalStruct(key string, output any, additionalDefaults []UnmarshalDefaults) error {
	refl.InitializeMapsAndSlices(output)
	finalSettings := mapx.NewMapX()

	var err error
	var ms *mapx.Struct
	var zeroSettings, defaults, settings *mapx.MapX

	if ms, err = c.buildMapStruct(output); err != nil {
		return fmt.Errorf("can not build mapx.Struct for output: %w", err)
	}

	if zeroSettings, defaults, err = ms.ReadZeroAndDefaultValues(); err != nil {
		return fmt.Errorf("can not read zeros and defaults for struct %T: %w", output, err)
	}

	finalSettings.Merge(".", zeroSettings)
	finalSettings.Merge(".", defaults)

	for _, def := range additionalDefaults {
		if err := def(c, finalSettings); err != nil {
			return fmt.Errorf("can not apply additional defaults: %w", err)
		}
	}

	if c.settings.Has(key) {
		if settings, err = c.settings.Get(key).Map(); err != nil {
			return fmt.Errorf("can not get settings for key: %s: %w", key, err)
		}

		finalSettings.Merge(".", settings)
	}

	environmentKey := c.resolveEnvKey(c.envKeyPrefix, key)
	environmentKeySettings, err := c.readEnvironmentFromStructKeys(environmentKey, ms.Keys())
	if err != nil {
		return fmt.Errorf("can not read environment key settings for key %s: %w", key, err)
	}

	environmentValueSettings, err := c.readEnvironmentFromValues(environmentKey, finalSettings)
	if err != nil {
		return fmt.Errorf("can not read environment value settings for key %s: %w", key, err)
	}

	finalSettings.Merge(".", environmentKeySettings)
	finalSettings.Merge(".", environmentValueSettings)

	c.settings.Set(key, finalSettings)

	if err = ms.Write(finalSettings); err != nil {
		return fmt.Errorf("error unmarshalling key: %s: %w", key, err)
	}

	validate := validator.New()
	err = validate.Struct(output)

	if err == nil {
		return nil
	}

	var invalidValidationError *validator.InvalidValidationError
	if errors.As(err, &invalidValidationError) {
		return fmt.Errorf("can not validate result of key: %s: %w", key, err)
	}

	errs := &multierror.Error{}
	for _, validationErr := range err.(validator.ValidationErrors) {
		err = fmt.Errorf("the setting %s with value %v does not match its requirement", validationErr.Field(), validationErr.Value())
		errs = multierror.Append(errs, err)
	}

	if errs != nil {
		return fmt.Errorf("validation failed for key: %s: %w", key, errs)
	}

	return nil
}
