// Package childenv builds the environment a child process receives.
//
// A child spawned with no explicit environment inherits its parent's entire
// environment, secrets included. Prumo is launched with model API keys, token
// stores and cloud credentials; an MCP server, an ACP agent or the opencode CLI
// it spawns therefore received every one of them, whatever that program's own
// permission policy says. A tool call that reads its own environment is a
// credential exfiltration with one command (GAP-144).
//
// The rule here is allowlist by default with a documented pass-through, not
// denylist. A denylist has to be right about every name a secret can arrive
// under, and the next provider brings a new one. What a child genuinely needs
// from the parent is PATH, HOME, locale, the terminal and the proxy settings;
// everything else has to be named on purpose.
package childenv

import (
	"os"
	"sort"
	"strings"
)

// passThroughPrefixes are variable namespaces a child legitimately inherits.
// They are namespaced because a whole namespace is a deliberate statement:
// these carry paths and locale, not credentials.
var passThroughPrefixes = []string{
	"PATH", "HOME", "LANG", "LC_", "TZ", "TERM", "TMPDIR", "TEMP", "TMP",
	"USER", "LOGNAME", "SHELL", "PWD", "XAUTHORITY", "XDG_",
	"HTTPS_PROXY", "HTTP_PROXY", "NO_PROXY", "SSL_CERT_FILE", "SSL_CERT_DIR",
	"NODE_EXTRA_CA_CERTS", "GIT_", "SSH_", // git and ssh config, not ssh keys
}

// deniedNames are refused even when a prefix would otherwise pass them through.
// SSH_AUTH_SOCK is an agent socket rather than a key file, but handing an
// unrelated child a usable agent socket is the same capability.
var deniedNames = map[string]bool{
	"SSH_AUTH_SOCK":         true,
	"SSH_AGENT_PID":         true,
	"GIT_ASKPASS":           true,
	"PRUMO_MODEL_API_KEY":   true,
	"PRUMO_API_KEY":         true,
	"ANTHROPIC_API_KEY":     true,
	"OPENAI_API_KEY":        true,
	"OPENROUTER_API_KEY":    true,
	"GOOGLE_API_KEY":        true,
	"GEMINI_API_KEY":        true,
	"AWS_SECRET_ACCESS_KEY": true,
	"AWS_SESSION_TOKEN":     true,
	"GITHUB_TOKEN":          true,
	"GH_TOKEN":              true,
	"NPM_TOKEN":             true,
	"DOCKER_PASSWORD":       true,
	"PGPASSWORD":            true,
	"PRIVATE_KEY":           true,
}

// allowedExact is the explicit pass-through for a name a specific deployment
// needs in a child. It is populated by the caller, not guessed here.
type Options struct {
	// Allow names variables to pass through even though they are not in the
	// pass-through prefixes. This is how a caller says "yes, this one". It does
	// not unlock the hard-denied names: those are the harness's own credentials
	// and reaching one is a statement, not a filter override.
	Allow []string
	// Extra variables to set on the child, overriding whatever survived.
	Extra map[string]string
}

// Build returns the environment for a child: the safe subset of the current
// process's environment, plus Extra, plus anything explicitly allowed.
//
// It takes no argument for the parent environment because there is only one to
// read, os.Environ. Tests that need a controlled parent use BuildFrom.
func Build(opts Options) []string {
	return BuildFrom(os.Environ(), opts)
}

// BuildFrom is Build with an explicit parent environment, so the rule can be
// tested without mutating the test process.
func BuildFrom(parent []string, opts Options) []string {
	allow := make(map[string]bool, len(opts.Allow))
	for _, name := range opts.Allow {
		allow[strings.ToUpper(strings.TrimSpace(name))] = true
	}

	seen := make(map[string]string)
	for _, entry := range parent {
		name, value, found := strings.Cut(entry, "=")
		if !found || name == "" {
			continue
		}
		upper := strings.ToUpper(name)
		// The hard-denied names are absolute for the pass-through path. Options
		// is about which inherited variables survive filtering; Extra is about
		// setting one deliberately. A credential the harness itself holds
		// belongs in the second category, where setting it is a statement.
		if deniedNames[upper] {
			continue
		}
		if !allow[upper] && !passesThrough(upper) {
			continue
		}
		seen[name] = value
	}
	for name, value := range opts.Extra {
		seen[name] = value
	}

	out := make([]string, 0, len(seen))
	for name, value := range seen {
		out = append(out, name+"="+value)
	}
	// Sorted, so a child sees a stable environment and a test can compare it.
	sort.Strings(out)
	return out
}

func passesThrough(upperName string) bool {
	for _, prefix := range passThroughPrefixes {
		if upperName == prefix || strings.HasPrefix(upperName, prefix) {
			return true
		}
	}
	return false
}

// RedactedPrefix is the value substituted for a secret found in text. It marks
// the place without revealing the value and without pretending the text is
// intact.
const RedactedPrefix = "[REDACTED_SECRET]"

// Redact scrubs known secret values out of child output. A child's stderr goes
// back into the harness and can reach the model and the run record, and a tool
// that prints its own environment prints the key with it.
func Redact(text string, secrets []string) string {
	for _, secret := range secrets {
		if len(secret) < 4 {
			// A short value would match ordinary text; redacting it would corrupt
			// the output for no gain, since it cannot be a real credential.
			continue
		}
		text = strings.ReplaceAll(text, secret, RedactedPrefix)
	}
	return text
}
