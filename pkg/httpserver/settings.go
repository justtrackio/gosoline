package httpserver

import "time"

// CompressionSettings control gzip support for requests and responses. By default, compressed requests are accepted and compressed responses are returned (if accepted by the client).
type CompressionSettings struct {
	Level         string `cfg:"level"         default:"default" validate:"oneof=none default best fast 0 1 2 3 4 5 6 7 8 9"`
	Decompression bool   `cfg:"decompression" default:"true"`
	// Exclude files by path, extension, or regular expression from being considered for compression. Useful if you are serving a format unknown to Gosoline.
	Exclude CompressionExcludeSettings `cfg:"exclude"`
}

// CompressionExcludeSettings allow enabling of gzip support.
type CompressionExcludeSettings struct {
	Extension []string `cfg:"extension"`
	Path      []string `cfg:"path"`
	PathRegex []string `cfg:"path_regex"`
}

type HealthCheckSettings struct {
	Port int    `cfg:"port" default:"8090"`
	Path string `cfg:"path" default:"/health"`
}

type ProfilingSettings struct {
	Enabled bool                 `cfg:"enabled" default:"false"`
	Api     ProfilingApiSettings `cfg:"api"`
}

type ProfilingApiSettings struct {
	Port int `cfg:"port" default:"8091"`
}

type Settings struct {
	// Port the API listens to.
	Port string `cfg:"port"        default:"8080"`
	// Mode is either debug, release, test.
	Mode string `cfg:"mode"        default:"release" validate:"oneof=release debug test"`
	// Compression settings.
	Compression CompressionSettings `cfg:"compression"`
	// Timeout settings.
	Timeout TimeoutSettings `cfg:"timeout"`
	// Logging settings
	Logging LoggingSettings `cfg:"logging"`
}

// TimeoutSettings configures IO timeouts.
type TimeoutSettings struct {
	// You need to give at least 1s as timeout.
	// Read timeout is the maximum duration for reading the entire request, including the body.
	Read time.Duration `cfg:"read"  default:"60s" validate:"min=1000000000"`
	// Write timeout is the maximum duration before timing out writes of the response.
	Write time.Duration `cfg:"write" default:"60s" validate:"min=1000000000"`
	// Idle timeout is the maximum amount of time to wait for the next request when keep-alives are enabled
	Idle time.Duration `cfg:"idle"  default:"60s" validate:"min=1000000000"`
}

type LoggingSettings struct {
	RequestBody       bool `cfg:"request_body"`
	RequestBodyBase64 bool `cfg:"request_body_base64"`
}
