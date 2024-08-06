package log

type HandlerIoWriterSettings struct {
	Level           string   `cfg:"level"            default:"info"`
	Channels        []string `cfg:"channels"`
	Formatter       string   `cfg:"formatter"        default:"console"`
	TimestampFormat string   `cfg:"timestamp_format" default:"15:04:05.000"`
	Writer          string   `cfg:"writer"           default:"stdout"`
}

type LoggerSettings struct {
	Handlers map[string]HandlerSettings `cfg:"handlers"`
}

type HandlerSettings struct {
	Type string `cfg:"type"`
}

type SentryHubSettings struct {
	Dsn         string
	Environment string
	AppName     string
}
