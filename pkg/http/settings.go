package http

import "time"

type CircuitBreakerSettings struct {
	Enabled          bool          `cfg:"enabled"           default:"false"`
	MaxFailures      int64         `cfg:"max_failures"      default:"10"`
	RetryDelay       time.Duration `cfg:"retry_delay"       default:"1m"`
	ExpectedStatuses []int         `cfg:"expected_statuses"`
}

type Settings struct {
	DisableCookies         bool                   `cfg:"disable_cookies"     default:"false"`
	FollowRedirects        bool                   `cfg:"follow_redirects"    default:"true"`
	RequestTimeout         time.Duration          `cfg:"request_timeout"     default:"30s"`
	RetryCount             int                    `cfg:"retry_count"         default:"5"`
	RetryMaxWaitTime       time.Duration          `cfg:"retry_max_wait_time" default:"2000ms"`
	RetryResetReaders      bool                   `cfg:"retry_reset_readers" default:"true"`
	RetryWaitTime          time.Duration          `cfg:"retry_wait_time"     default:"100ms"`
	CircuitBreakerSettings CircuitBreakerSettings `cfg:"circuit_breaker"`
	TransportSettings      TransportSettings      `cfg:"transport"`
}

type TransportSettings struct {
	// TLSHandshakeTimeout specifies the maximum amount of time to
	// wait for a TLS handshake. Zero means no timeout.
	TLSHandshakeTimeout time.Duration `cfg:"tls_handshake_timeout" default:"10s"`

	// DisableKeepAlives, if true, disables HTTP keep-alives and
	// will only use the connection to the server for a single
	// HTTP request.
	//
	// This is unrelated to the similarly named TCP keep-alives.
	DisableKeepAlives bool `cfg:"disable_keep_alives" default:"false"`

	// DisableCompression, if true, prevents the Transport from
	// requesting compression with an "Accept-Encoding: gzip"
	// request header when the Request contains no existing
	// Accept-Encoding value. If the Transport requests gzip on
	// its own and gets a gzipped response, it's transparently
	// decoded in the Response.Body. However, if the user
	// explicitly requested gzip it is not automatically
	// uncompressed.
	DisableCompression bool `cfg:"disable_compression" default:"false"`

	// MaxIdleConns controls the maximum number of idle (keep-alive)
	// connections across all hosts. Zero means no limit.
	MaxIdleConns int `cfg:"max_idle_conns" default:"100"`

	// MaxIdleConnsPerHost, if non-zero, controls the maximum idle
	// (keep-alive) connections to keep per-host. If zero,
	// GOMAXPROCS+1 is used.
	MaxIdleConnsPerHost int `cfg:"max_idle_conns_per_host" default:"0"`

	// MaxConnsPerHost optionally limits the total number of
	// connections per host, including connections in the dialing,
	// active, and idle states. On limit violation, dials will block.
	//
	// Zero means no limit.
	MaxConnsPerHost int `cfg:"max_conns_per_host" default:"0"`

	// IdleConnTimeout is the maximum amount of time an idle
	// (keep-alive) connection will remain idle before closing
	// itself.
	// Zero means no limit.
	IdleConnTimeout time.Duration `cfg:"idle_conn_timeout" default:"90s"`

	// ResponseHeaderTimeout, if non-zero, specifies the amount of
	// time to wait for a server's response headers after fully
	// writing the request (including its body, if any). This
	// time does not include the time to read the response body.
	ResponseHeaderTimeout time.Duration `cfg:"response_header_timeout" default:"0s"`

	// ExpectContinueTimeout, if non-zero, specifies the amount of
	// time to wait for a server's first response headers after fully
	// writing the request headers if the request has an
	// "Expect: 100-continue" header. Zero means no timeout and
	// causes the body to be sent immediately, without
	// waiting for the server to approve.
	// This time does not include the time to send the request header.
	ExpectContinueTimeout time.Duration `cfg:"expect_continue_timeout" default:"1s"`

	// MaxResponseHeaderBytes specifies a limit on how many
	// response bytes are allowed in the server's response
	// header.
	//
	// Zero means to use a default limit.
	MaxResponseHeaderBytes int64 `cfg:"max_response_header_bytes" default:"0"`

	// WriteBufferSize specifies the size of the write buffer used
	// when writing to the transport.
	// If zero, a default (currently 4KB) is used.
	WriteBufferSize int `cfg:"write_buffer_size" default:"0"`

	// ReadBufferSize specifies the size of the read buffer used
	// when reading from the transport.
	// If zero, a default (currently 4KB) is used.
	ReadBufferSize int `cfg:"read_buffer_size" default:"0"`

	DialerSettings DialerSettings `cfg:"dialer"`
}

type DialerSettings struct {
	// KeepAlive specifies the interval between keep-alive
	// probes for an active network connection.
	// If zero, keep-alive probes are sent with a default value
	// (currently 15 seconds), if supported by the protocol and operating
	// system. Network protocols or operating systems that do
	// not support keep-alives ignore this field.
	// If negative, keep-alive probes are disabled.
	KeepAlive time.Duration `cfg:"keep_alive" default:"30s"`

	// Timeout is the maximum amount of time a dial will wait for
	// a connect to complete. If Deadline is also set, it may fail
	// earlier.
	//
	// The default is no timeout.
	//
	// When using TCP and dialing a host name with multiple IP
	// addresses, the timeout may be divided between them.
	//
	// With or without a timeout, the operating system may impose
	// its own earlier timeout. For instance, TCP timeouts are
	// often around 3 minutes.
	Timeout time.Duration `cfg:"timeout" default:"30s"`

	// DualStack previously enabled RFC 6555 Fast Fallback
	// support, also known as "Happy Eyeballs", in which IPv4 is
	// tried soon if IPv6 appears to be misconfigured and
	// hanging.
	//
	// Deprecated: Fast Fallback is enabled by default. To
	// disable, set FallbackDelay to a negative value.
	DualStack bool `cfg:"dual_stack" default:"true"`

	// FallbackDelay specifies the length of time to wait before
	// spawning a RFC 6555 Fast Fallback connection. That is, this
	// is the amount of time to wait for IPv6 to succeed before
	// assuming that IPv6 is misconfigured and falling back to
	// IPv4.
	//
	// If zero, a default delay of 300ms is used.
	// A negative value disables Fast Fallback support.
	FallbackDelay time.Duration `cfg:"fallback_delay" default:"0s"`
}
