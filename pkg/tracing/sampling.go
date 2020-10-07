package tracing

type SamplingConfiguration struct {
	Version int          `cfg:"version" default:"1"`
	Default SampleRule   `cfg:"default"`
	Rules   []SampleRule `cfg:"rules"`
}

type SampleRule struct {
	Description string  `cfg:"description" default:"default"`
	ServiceName string  `cfg:"service_name" default:"*"`
	HttpMethod  string  `cfg:"http_method" default:"*"`
	UrlPath     string  `cfg:"url_path" default:"*"`
	FixedTarget uint64  `cfg:"fixed_target" default:"1"`
	Rate        float64 `cfg:"rate" default:"0.05"`
}
