package grpcserver

// Settings of the Server.
type Settings struct {
	// Port where the grpc.Server will be listening to.
	Port string `cfg:"port" default:"8081"`
	// Health related settings.
	Health Health `cfg:"health"`
	// Statistics related settings.
	Stats Stats `cfg:"stats"`
}

// Health settings of the Server.
type Health struct {
	// Enabled determines if the default health.checks will be used
	// the default health-checks utilizes the google.golang.org/grpc/health/grpc_health_v1 pkg.
	Enabled bool `cfg:"enabled" default:"true"`
}

type Stats struct {
	// Enabled defines if the statistics handler is enabled.
	Enabled bool `cfg:"enabled" default:"true"`
	// LogLevel defines the log level for the statistics logs
	LogLevel string `cfg:"log_level" default:"debug" validate:"oneof=debug info"`
	// LogPayload defines whether to log the incoming and outgoing payloads of a gRPC method.
	LogPayload bool `cfg:"log_payload" default:"false"`
	// LogData defines whether to log the incoming and outgoing raw data of a gRPC method.
	LogData bool `cfg:"log_data" default:"false"`
	// Channel to log the statistics to.
	Channel string `cfg:"channel" default:"grpc_stats"`
}
