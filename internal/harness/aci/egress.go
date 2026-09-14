// Egress policy for network-capable local execution (GAP-004 remainder).
// The container executor denies network structurally; this policy covers
// the local executor. Nil policy preserves legacy unrestricted behavior
// (documented posture); --egress-deny flips to fail-closed with an
// allowlist. Destination rules reuse internal/egress data classes.
package aci

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/raillen/prumo/internal/egress"
)

// EgressPolicy gates outbound hosts for local command tools.
type EgressPolicy struct {
	DefaultDeny bool
	AllowHosts  []string // exact or ".suffix" matches
	DataClass   egress.Class
}

var urlRe = regexp.MustCompile(`(https?)://([^/\s:@]+)`)

// endpoint is one referenced network destination.
type endpoint struct {
	scheme string
	host   string
}

// ExtractEndpoints lists URL destinations referenced by a shell command.
func ExtractEndpoints(command string) []endpoint {
	seen := map[string]bool{}
	var out []endpoint
	for _, m := range urlRe.FindAllStringSubmatch(command, -1) {
		host := strings.ToLower(strings.Trim(m[2], "."))
		if host == "" || seen[host] {
			continue
		}
		seen[host] = true
		out = append(out, endpoint{scheme: m[1], host: host})
	}
	return out
}

func hostAllowed(host string, allow []string) bool {
	for _, a := range allow {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" {
			continue
		}
		if strings.HasPrefix(a, ".") {
			if host == a[1:] || strings.HasSuffix(host, a) {
				return true
			}
			continue
		}
		if host == a {
			return true
		}
	}
	return false
}

// CheckEgress enforces the policy for one command. Nil policy allows all
// (legacy posture, explicit). Failures name the host and the fix.
func CheckEgress(p *EgressPolicy, command string) error {
	if p == nil {
		return nil
	}
	class := p.DataClass
	if class == "" {
		class = egress.Internal
	}
	for _, ep := range ExtractEndpoints(command) {
		trusted := hostAllowed(ep.host, p.AllowHosts)
		if p.DefaultDeny && !trusted {
			return fmt.Errorf("egress denied to %s (not in allowlist; use --egress-allow)", ep.host)
		}
		secure := ep.scheme == "https"
		if d := egress.AllowEgress(class, ep.host, secure, trusted); !d.Allowed {
			return fmt.Errorf("egress denied to %s: %s", ep.host, d.Reason)
		}
	}
	return nil
}
