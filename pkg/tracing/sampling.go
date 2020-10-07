package tracing

type SamplingConfiguration struct {
	Version int          `json:"version" cfg:"version" default:"1"`
	Default SampleRule   `json:"default" cfg:"default"`
	Rules   []SampleRule `json:"rules" cfg:"rules"`
}

type SampleRule struct {
	Description string  `json:"description" cfg:"description" default:"default"`
	ServiceName string  `json:"service_name" cfg:"service_name"`
	HttpMethod  string  `json:"http_method" cfg:"http_method"`
	UrlPath     string  `json:"url_path" cfg:"url_path"`
	FixedTarget uint64  `json:"fixed_target" cfg:"fixed_target" default:"1"`
	Rate        float64 `json:"rate" cfg:"rate" default:"0.05"`
}
