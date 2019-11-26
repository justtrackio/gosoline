package cfg

import "strings"

type Option func(cfg *config) error

func WithConfigFile(filePath string, fileType string) Option {
	return func(cfg *config) error {
		return readConfigFromFile(cfg, filePath, fileType)
	}
}

func WithConfigMap(settings map[string]interface{}) Option {
	return func(cfg *config) error {
		return cfg.mergeSettings(settings)
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

func WithErrorHandlers(handlers ...ErrorHandler) Option {
	return func(cfg *config) error {
		cfg.errorHandlers = handlers

		return nil
	}
}

func WithSanitizers(sanitizer ...Sanitizer) Option {
	return func(cfg *config) error {
		cfg.sanitizers = append(cfg.sanitizers, sanitizer...)

		return nil
	}
}
