package http_test

import (
	"net"
	"testing"

	"github.com/justtrackio/gosoline/pkg/funk"
	"github.com/justtrackio/gosoline/pkg/http"
	httpMocks "github.com/justtrackio/gosoline/pkg/http/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func ipAddresses(ips ...net.IP) []net.IPAddr {
	return funk.Map(ips, func(ip net.IP) net.IPAddr {
		return net.IPAddr{IP: ip}
	})
}

func ipsOf(addresses []net.IPAddr) []net.IP {
	return funk.Map(addresses, func(addr net.IPAddr) net.IP {
		return addr.IP
	})
}

func TestUrlValidator_Validate(t *testing.T) {
	publicIp := net.ParseIP("93.184.216.34")

	tests := []struct {
		name string
		url  string
		host string
		ips  []net.IP
		// resolve is true when the validator is expected to look the host up
		resolve bool
		wantErr bool
		wantIps []net.IP
	}{
		{
			name:    "rejects a non-http scheme",
			url:     "ftp://example.com/file",
			wantErr: true,
		},
		{
			name:    "rejects a non-standard port",
			url:     "http://example.com:8080",
			wantErr: true,
		},
		{
			name:    "rejects a bare ip address",
			url:     "http://93.184.216.34",
			wantErr: true,
		},
		{
			name:    "rejects a host resolving to a private ip",
			url:     "http://example.com",
			host:    "example.com",
			ips:     []net.IP{net.ParseIP("10.0.0.1")},
			resolve: true,
			wantErr: true,
		},
		{
			name:    "rejects a host resolving to loopback",
			url:     "http://example.com",
			host:    "example.com",
			ips:     []net.IP{net.ParseIP("127.0.0.1")},
			resolve: true,
			wantErr: true,
		},
		{
			name:    "rejects a host resolving to a mapped private ipv4",
			url:     "http://example.com",
			host:    "example.com",
			ips:     []net.IP{net.ParseIP("::ffff:10.0.0.1")},
			resolve: true,
			wantErr: true,
		},
		{
			name:    "rejects a host resolving to a nat64 address",
			url:     "http://example.com",
			host:    "example.com",
			ips:     []net.IP{net.ParseIP("64:ff9b::a00:1")},
			resolve: true,
			wantErr: true,
		},
		{
			name:    "rejects a host resolving to the broadcast address",
			url:     "http://example.com",
			host:    "example.com",
			ips:     []net.IP{net.ParseIP("255.255.255.255")},
			resolve: true,
			wantErr: true,
		},
		{
			name:    "rejects a host without any ip address",
			url:     "http://example.com",
			host:    "example.com",
			ips:     []net.IP{},
			resolve: true,
			wantErr: true,
		},
		{
			name:    "allows a host resolving to a public ip",
			url:     "http://example.com",
			host:    "example.com",
			ips:     []net.IP{publicIp},
			resolve: true,
			wantIps: []net.IP{publicIp},
		},
		{
			name:    "allows https on the default port",
			url:     "https://example.com",
			host:    "example.com",
			ips:     []net.IP{publicIp},
			resolve: true,
			wantIps: []net.IP{publicIp},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := httpMocks.NewResolver(t)
			if tt.resolve {
				resolver.EXPECT().LookupIPAddr(mock.Anything, tt.host).Return(ipAddresses(tt.ips...), nil)
			}

			validator := http.NewUrlValidatorWithInterfaces(resolver)
			ips, err := validator.Validate(t.Context(), tt.url)

			if tt.wantErr {
				assert.Error(t, err)

				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantIps, ipsOf(ips))
		})
	}
}
