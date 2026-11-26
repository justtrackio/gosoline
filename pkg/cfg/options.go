package cfg

import (
	"flag"
	"os"
	"strings"
)

type Option func(cfg *config) error

type configFiles []string

func (c *configFiles) String() string {
	return strings.Join(*c, ",")
}

func (c *configFiles) Set(value string) error {
	*c = append(*c, value)

	return nil
}

func WithConfigBytes(bytes []byte, format string) Option {
	return func(cfg *config) error {
		return readConfigFromBytes(cfg, bytes, format)
	}
}

func WithConfigFile(filePath string, fileType string) Option {
	return func(cfg *config) error {
		return readConfigFromFile(cfg, filePath, fileType)
	}
}

func WithConfigFileFlag(flagName string) Option {
	return func(cfg *config) error {
		flags := flag.NewFlagSet("cfg", flag.ContinueOnError)

		var configFiles configFiles
		flags.Var(&configFiles, flagName, "path to a config file")
		err := flags.Parse(os.Args[1:])
		if err != nil {
			return err
		}

		for _, configFile := range configFiles {
			err := readConfigFromFile(cfg, configFile, "yml")
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func WithConfigMap(settings map[string]any, mergeOptions ...MergeOption) Option {
	return func(cfg *config) error {
		return cfg.merge(".", settings, mergeOptions...)
	}
}

func WithConfigSetting(key string, settings any, mergeOptions ...MergeOption) Option {
	return func(cfg *config) error {
		return cfg.merge(key, settings, mergeOptions...)
	}
}

// WithConfigOverride sets a string value that takes precedence over environment variables.
// It affects this configuration only; process environment variables remain unchanged.
func WithConfigOverride(key string, value string) Option {
	return func(cfg *config) error {
		if err := cfg.merge(key, value); err != nil {
			return err
		}
		cfg.overrides.Set(key, cfg.settings.Get(key).Data())

		return nil
	}
}

func WithEnvKeyPrefix(prefix string) Option {
	return func(cfg *config) error {
		cfg.envKeyPrefix = prefix

		return nil
	}
}

func WithEnvKeyReplacer(replacer *strings.Replacer) Option {
	return func(cfg *config) error {
		cfg.envKeyReplacer = replacer

		return nil
	}
}

func WithSanitizers(sanitizer ...Sanitizer) Option {
	return func(cfg *config) error {
		cfg.sanitizers = append(cfg.sanitizers, sanitizer...)

		return nil
	}
}
