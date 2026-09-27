package model

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// A model provider's base URL arrives from a request or from the environment,
// and the harness then sends the conversation — and an API key — to whatever
// address it names. That made it a server-side request forgery primitive: point
// it at the cloud metadata endpoint, or at a service bound to loopback that the
// daemon can reach but the caller could not, and the daemon fetched it on the
// caller's behalf (GAP-111).
//
// The check is made in the dialer rather than once at construction, on purpose.
// Validating the URL before the request catches a literal private address;
// resolving the name separately and then connecting leaves a window in which
// DNS answers the check with one address and the connection with another. One
// resolution, checked, is the only form that closes the window.

// DestinationPolicy says which network a provider may be reached on.
type DestinationPolicy struct {
	// AllowLoopback permits 127.0.0.0/8 and ::1. Needed for a local gateway
	// (llama.cpp, ollama, a test server), and off by default because loopback
	// is exactly what an SSRF probe wants.
	AllowLoopback bool
	// AllowPrivate permits RFC1918, unique-local and link-local ranges. Off by
	// default: a provider has no reason to live on the office LAN.
	AllowPrivate bool
	// AllowMetadata permits the cloud metadata endpoints. Off by default, and
	// the one thing worth naming out loud: these hand out credentials to
	// anything that can make an HTTP request from the host.
	AllowMetadata bool
	// AllowPlainHTTP permits http://. Off by default, so credentials are not
	// sent in the clear.
	AllowPlainHTTP bool
	// AllowedHosts, when non-empty, restricts connections to these hostnames.
	// Checked after resolution, so a name in the list cannot be used to smuggle
	// a private address in.
	AllowedHosts []string
}

// DefaultDestinationPolicy is what a daemon serving more than one user gets.
// It is the reason the base URL cannot be treated as trusted input.
func DefaultDestinationPolicy() DestinationPolicy {
	return DestinationPolicy{}
}

// LocalDevelopmentDestinationPolicy allows a loopback gateway over plain HTTP.
// It exists for a human running a model locally; it is opt-in, never the
// default, and never applied to a request the daemon received from a client.
func LocalDevelopmentDestinationPolicy() DestinationPolicy {
	return DestinationPolicy{AllowLoopback: true, AllowPlainHTTP: true}
}

func (p DestinationPolicy) allowsHost(host string) bool {
	if len(p.AllowedHosts) == 0 {
		return true
	}
	for _, allowed := range p.AllowedHosts {
		if strings.EqualFold(allowed, host) {
			return true
		}
	}
	return false
}

// checkAddr classifies one resolved address against the policy.
func (p DestinationPolicy) checkAddr(addr netip.Addr) error {
	if !addr.IsValid() {
		return fmt.Errorf("model destination resolved to an invalid address")
	}
	if addr.IsLoopback() {
		if p.AllowLoopback {
			return nil
		}
		return fmt.Errorf("model destination resolves to the loopback address %s; loopback is not allowed", addr)
	}
	if IsMetadataAddr(addr) {
		if p.AllowMetadata {
			return nil
		}
		return fmt.Errorf("model destination resolves to the cloud metadata address %s", addr)
	}
	if IsLinkLocal(addr) {
		return fmt.Errorf("model destination resolves to the link-local address %s", addr)
	}
	if IsPrivate(addr) {
		if p.AllowPrivate {
			return nil
		}
		return fmt.Errorf("model destination resolves to the private address %s", addr)
	}
	return nil
}

// ValidateDestinationURL checks a base URL before it is used. It catches the
// literal cases — a private address typed straight into the config, a file:
// scheme, a missing host. The dialer still runs for every request, because a
// hostname can resolve to a private address and this check does not resolve.
func ValidateDestinationURL(raw string, policy DestinationPolicy) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fmt.Errorf("model destination is empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("model destination is not a valid URL: %w", err)
	}
	switch parsed.Scheme {
	case "https":
	case "http":
		if !policy.AllowPlainHTTP {
			return fmt.Errorf("model destination must use https; %s uses plain http", parsed.Host)
		}
	default:
		return fmt.Errorf("model destination scheme %q is not supported; use https", parsed.Scheme)
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("model destination has no host")
	}
	if !policy.allowsHost(host) {
		return fmt.Errorf("model destination host %q is not in the allowed hosts", host)
	}
	// Some names mean an internal address by definition, so they can be refused
	// now instead of at the first dial. The dialer still covers every other name.
	switch strings.ToLower(strings.TrimSuffix(host, ".")) {
	case "localhost", "ip6-localhost", "ip6-loopback":
		if !policy.AllowLoopback {
			return fmt.Errorf("model destination host %q is loopback; loopback is not allowed", host)
		}
	case "metadata.google.internal", "metadata.goog", "instance-data", "instance-data.ec2.internal":
		if !policy.AllowMetadata {
			return fmt.Errorf("model destination host %q is a cloud metadata endpoint", host)
		}
	}
	// A literal address can be judged now. A name is judged by the dialer.
	if addr, parseErr := netip.ParseAddr(host); parseErr == nil {
		return policy.checkAddr(addr)
	}
	return nil
}

// NewDestinationHTTPClient returns a client whose dialer resolves the host once
// and checks every address it gets before connecting.
func NewDestinationHTTPClient(policy DestinationPolicy) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		ForceAttemptHTTP2:   true,
		MaxIdleConns:        8,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, splitErr := net.SplitHostPort(addr)
			if splitErr != nil {
				return nil, fmt.Errorf("model destination %q is malformed: %w", addr, splitErr)
			}
			if !policy.allowsHost(host) {
				return nil, fmt.Errorf("model destination host %q is not in the allowed hosts", host)
			}
			// One resolution, used for the check and the connection.
			ips, resolveErr := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if resolveErr != nil {
				return nil, fmt.Errorf("resolve model destination %s: %w", host, resolveErr)
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("model destination %s resolved to no address", host)
			}
			var lastErr error
			for _, ip := range ips {
				if checkErr := policy.checkAddr(ip); checkErr != nil {
					lastErr = checkErr
					continue
				}
				conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if dialErr == nil {
					return conn, nil
				}
				lastErr = dialErr
			}
			if lastErr == nil {
				lastErr = fmt.Errorf("model destination %s has no permitted address", host)
			}
			return nil, lastErr
		},
	}
	return &http.Client{Transport: transport, Timeout: 120 * time.Second}
}

// IsPrivate reports whether addr is in a range reserved for internal networks:
// RFC1918, carrier-grade NAT and IPv6 unique-local.
func IsPrivate(addr netip.Addr) bool {
	return addr.IsPrivate()
}

// IsLinkLocal reports whether addr is link-local, which covers 169.254.0.0/16
// on IPv4 and fe80::/10 on IPv6.
func IsLinkLocal(addr netip.Addr) bool {
	if addr.Is4() {
		return addr.As4()[0] == 169 && addr.As4()[1] == 254
	}
	return addr.IsLinkLocalUnicast()
}

// metadataV4 is the address every major cloud exposes instance credentials on.
var metadataV4 = netip.MustParseAddr("169.254.169.254")

// metadataV6 is the IPv6 form of the same endpoint.
var metadataV6 = netip.MustParseAddr("fd00:ec2::254")

// IsMetadataAddr reports whether addr is a known cloud metadata endpoint. The
// link-local check catches most of them already; this names the two that matter
// so the refusal says why rather than only "private".
func IsMetadataAddr(addr netip.Addr) bool {
	return addr == metadataV4 || addr == metadataV6
}
