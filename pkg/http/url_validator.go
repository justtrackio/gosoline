package http

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"slices"

	"github.com/justtrackio/gosoline/pkg/funk"
)

// restrictedIpBlocks are the ip ranges a client id metadata document must not point at. This service fetches a document
// from a caller supplied url while running inside the cluster, so anything reachable there is exposed; pointing the
// fetch at an internal address would leak it.
var restrictedIpBlocks = []net.IPNet{
	// Private IPv4 ranges
	{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)},
	{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)},
	{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)},

	// Carrier-grade NAT
	{IP: net.ParseIP("100.64.0.0"), Mask: net.CIDRMask(10, 32)},

	// Link-local IPv4
	{IP: net.ParseIP("169.254.0.0"), Mask: net.CIDRMask(16, 32)},

	// Unspecified and broadcast addresses
	{IP: net.ParseIP("0.0.0.0"), Mask: net.CIDRMask(32, 32)},
	{IP: net.ParseIP("255.255.255.255"), Mask: net.CIDRMask(32, 32)},
	{IP: net.ParseIP("::"), Mask: net.CIDRMask(128, 128)},

	// Private (unique local) and link-local IPv6
	{IP: net.ParseIP("fc00::"), Mask: net.CIDRMask(7, 128)},
	{IP: net.ParseIP("fe80::"), Mask: net.CIDRMask(10, 128)},

	// NAT64: the address embeds an IPv4 host that a NAT64 translator can map to
	// an internal address, so it must not be reachable through a public name.
	{IP: net.ParseIP("64:ff9b::"), Mask: net.CIDRMask(96, 128)},

	// Loopback
	{IP: net.ParseIP("127.0.0.0"), Mask: net.CIDRMask(8, 32)},
	{IP: net.ParseIP("::1"), Mask: net.CIDRMask(128, 128)},
}

// Resolver resolves hostnames to ip addresses.
//
//go:generate go run github.com/vektra/mockery/v2 --name Resolver
type Resolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// A UrlValidator checks if a URL can safely be called, meaning it does not:
//   - point to an arbitrary IP address
//   - resolve to an IP inside our private network
//   - point to a non-standard port
//
//go:generate go run github.com/vektra/mockery/v2 --name UrlValidator
type UrlValidator interface {
	// Validate reports whether the url may be fetched. It rejects anything which is not a plain http(s) url on a standard
	// port whose host resolves to a public address. It returns a list of validated IP addresses the URL was resolved to.
	// The caller is responsible for resolving the hostname to only those IPs when performing a request to the given URL.
	Validate(ctx context.Context, rawUrl string) (ips []net.IPAddr, err error)
}

type urlValidator struct {
	resolver Resolver
}

type nopUrlValidator struct{}

func NewNopUrlValidator() UrlValidator {
	return nopUrlValidator{}
}

func (nopUrlValidator) Validate(context.Context, string) ([]net.IPAddr, error) {
	return nil, nil
}

func NewUrlValidator() UrlValidator {
	return NewUrlValidatorWithInterfaces(net.DefaultResolver)
}

func NewUrlValidatorWithInterfaces(resolver Resolver) UrlValidator {
	return &urlValidator{
		resolver: resolver,
	}
}

func (v *urlValidator) Validate(ctx context.Context, rawUrl string) (ips []net.IPAddr, err error) {
	parsed, err := url.Parse(rawUrl)
	if err != nil {
		return nil, fmt.Errorf("parsing the url %q: %w", rawUrl, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("the url %q uses the unsupported scheme %q", rawUrl, parsed.Scheme)
	}

	host := parsed.Hostname()
	if err := checkHostAndPort(rawUrl, parsed, host); err != nil {
		return nil, err
	}

	// the host has to be resolved and its addresses checked, otherwise a caller could point a public domain at an
	// internal address and the fetch would reach it
	ips, err = v.resolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolving the host %q: %w", host, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP addresses found for host %q", host)
	}

	if restricted, ok := firstRestrictedIp(ips); ok {
		return nil, fmt.Errorf("the host %q resolves to the restricted address %s", host, restricted.IP)
	}

	return ips, nil
}

// checkHostAndPort rejects urls that use a non-standard port or a bare ip
// address.
func checkHostAndPort(rawUrl string, parsed *url.URL, host string) error {
	if port := parsed.Port(); port != "" && !isStandardPort(parsed.Scheme, port) {
		return fmt.Errorf("the url %q uses the unsupported port %q", rawUrl, port)
	}

	if net.ParseIP(host) != nil {
		return fmt.Errorf("the url %q must not use a bare ip address", rawUrl)
	}

	return nil
}

func isStandardPort(scheme, port string) bool {
	return (scheme == "http" && port == "80") || (scheme == "https" && port == "443")
}

// firstRestrictedIp returns the first address in addresses that is inside a
// restricted range.
func firstRestrictedIp(addresses []net.IPAddr) (net.IPAddr, bool) {
	return funk.FindFirstFunc(addresses, isRestrictedIp)
}

func isRestrictedIp(addr net.IPAddr) bool {
	return slices.ContainsFunc(restrictedIpBlocks, func(block net.IPNet) bool {
		return block.Contains(addr.IP)
	})
}
