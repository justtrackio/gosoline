package http

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	netUrl "net/url"
	"time"
)

type (
	DialerOption    func(dialer *net.Dialer) error
	TransportOption func(transport *http.Transport) error
)

type Option struct {
	DialerOption    DialerOption
	TransportOption TransportOption
}

func WithDialerOption(dialer DialerOption) Option {
	return Option{
		DialerOption: dialer,
	}
}

func WithTransportOption(transport TransportOption) Option {
	return Option{
		TransportOption: transport,
	}
}

func partitionOptions(options []Option) (dialerOptions []DialerOption, transportOptions []TransportOption) {
	for _, option := range options {
		if option.DialerOption != nil {
			dialerOptions = append(dialerOptions, option.DialerOption)
		}
		if option.TransportOption != nil {
			transportOptions = append(transportOptions, option.TransportOption)
		}
	}

	return dialerOptions, transportOptions
}

func withTLSConfig(mutate func(tlsConfig *tls.Config) error) TransportOption {
	return func(transport *http.Transport) error {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}

		return mutate(transport.TLSClientConfig)
	}
}

// WithRootCAs completely replaces the pool of root CAs with the given one.
// To just add a singular CA, consider using WithAdditionalRootCAFromPEM instead.
func WithRootCAs(rootCAs *x509.CertPool) Option {
	return WithTransportOption(withTLSConfig(func(tlsConfig *tls.Config) error {
		tlsConfig.RootCAs = rootCAs

		return nil
	}))
}

// WithAdditionalRootCAFromPEM appends the PEM encoded certificates in pemData
// to the system certificate pool and uses the result as the RootCAs of the
// client. Use this to trust additional, e.g. internal, certificate authorities
// without replacing the system defaults.
func WithAdditionalRootCAFromPEM(pemData []byte) Option {
	return WithTransportOption(withTLSConfig(func(tlsConfig *tls.Config) error {
		rootCAs, err := x509.SystemCertPool()
		if err != nil {
			rootCAs = x509.NewCertPool()
		}

		if ok := rootCAs.AppendCertsFromPEM(pemData); !ok {
			return fmt.Errorf("no certificates found in pem data")
		}

		tlsConfig.RootCAs = rootCAs

		return nil
	}))
}

// WithInsecureSkipVerify controls whether the client verifies the server's TLS
// certificates. If skip is true, certificate expiry, hostname matching and
// unknown certificate authorities are all ignored, which exposes the
// connection to man-in-the-middle attacks.
//
// WARNING: Only use this in development or test environments, or against
// trusted networks with self-signed certificates. If possible, consider using
// WithAdditionalRootCAFromPEM instead to add your custom CA instead.
func WithInsecureSkipVerify(skip bool) Option {
	return WithTransportOption(withTLSConfig(func(tlsConfig *tls.Config) error {
		tlsConfig.InsecureSkipVerify = skip

		return nil
	}))
}

// WithClientCertificates configures the client certificates used for mutual
// TLS authentication.
func WithClientCertificates(certs ...tls.Certificate) Option {
	return WithTransportOption(withTLSConfig(func(tlsConfig *tls.Config) error {
		tlsConfig.Certificates = append(tlsConfig.Certificates, certs...)

		return nil
	}))
}

// WithServerName overrides the name used to authenticate the remote server and
// included in the Server Name Indication (SNI) extension of the TLS handshake.
func WithServerName(name string) Option {
	return WithTransportOption(withTLSConfig(func(tlsConfig *tls.Config) error {
		tlsConfig.ServerName = name

		return nil
	}))
}

// WithMinTLSVersion sets the minimum TLS version the client accepts, e.g.
// tls.VersionTLS13.
func WithMinTLSVersion(version uint16) Option {
	return WithTransportOption(withTLSConfig(func(tlsConfig *tls.Config) error {
		tlsConfig.MinVersion = version

		return nil
	}))
}

// WithProxy sets the proxy function of the transport. Use a nil proxy to
// disable proxying.
func WithProxy(proxy func(request *http.Request) (*netUrl.URL, error)) Option {
	return WithTransportOption(func(transport *http.Transport) error {
		transport.Proxy = proxy

		return nil
	})
}

// WithForceHTTP2 controls whether the client attempts to use HTTP/2 without
// explicit registration.
func WithForceHTTP2(force bool) Option {
	return WithTransportOption(func(transport *http.Transport) error {
		transport.ForceAttemptHTTP2 = force

		return nil
	})
}

// WithDialerTimeout sets the maximum amount of time a dial will wait for a
// connection to complete.
func WithDialerTimeout(timeout time.Duration) Option {
	return WithDialerOption(func(dialer *net.Dialer) error {
		dialer.Timeout = timeout

		return nil
	})
}

// WithDialerKeepAlive sets the interval between keep-alive probes for an
// active network connection.
func WithDialerKeepAlive(keepAlive time.Duration) Option {
	return WithDialerOption(func(dialer *net.Dialer) error {
		dialer.KeepAlive = keepAlive

		return nil
	})
}
