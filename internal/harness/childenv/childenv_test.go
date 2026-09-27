package childenv

import (
	"strings"
	"testing"
)

// The failure this package exists to prevent: a child that can read a key out
// of its own environment. Every test here is about what a child does NOT see.

func TestSecretsAreNotPassedToAChild(t *testing.T) {
	parent := []string{
		"PATH=/usr/bin",
		"HOME=/home/user",
		"PRUMO_MODEL_API_KEY=sk-secret-value",
		"ANTHROPIC_API_KEY=sk-ant-value",
		"OPENAI_API_KEY=sk-openai-value",
		"AWS_SECRET_ACCESS_KEY=wJalr-value",
		"GITHUB_TOKEN=ghp_value",
		"SSH_AUTH_SOCK=/tmp/ssh-agent.sock",
	}
	got := strings.Join(BuildFrom(parent, Options{}), "\n")
	for _, secret := range []string{
		"sk-secret-value", "sk-ant-value", "sk-openai-value",
		"wJalr-value", "ghp_value", "/tmp/ssh-agent.sock",
	} {
		if strings.Contains(got, secret) {
			t.Errorf("a child must not receive %q; env was:\n%s", secret, got)
		}
	}
}

func TestUnrecognisedSecretsAreNotPassedToAChild(t *testing.T) {
	// The point of an allowlist rather than a denylist: the next provider brings
	// a name nobody has seen yet, and it still must not leak.
	parent := []string{
		"PATH=/usr/bin",
		"SOME_VENDOR_TOKEN=super-secret",
		"MY_COMPANY_CREDENTIAL=also-secret",
		"AWS_SESSION_TOKEN=session-secret",
		"PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----",
	}
	got := strings.Join(BuildFrom(parent, Options{}), "\n")
	for _, secret := range []string{"super-secret", "also-secret", "session-secret", "BEGIN RSA"} {
		if strings.Contains(got, secret) {
			t.Errorf("an unrecognised secret must not reach a child: %q\n%s", secret, got)
		}
	}
}

func TestWhatAChildGenuinelyNeedsIsKept(t *testing.T) {
	// An allowlist that drops PATH breaks every child process, so this is the
	// other half of the property: what is kept is what is needed.
	parent := []string{
		"PATH=/usr/bin:/bin",
		"HOME=/home/user",
		"LANG=en_US.UTF-8",
		"TERM=xterm-256color",
		"TMPDIR=/tmp",
		"HTTPS_PROXY=http://proxy:3128",
		"GIT_SSH_COMMAND=ssh",
		"XDG_CONFIG_HOME=/home/user/.config",
	}
	got := BuildFrom(parent, Options{})
	joined := strings.Join(got, "\n")
	for _, keep := range []string{
		"PATH=/usr/bin:/bin", "HOME=/home/user", "LANG=en_US.UTF-8",
		"TERM=xterm-256color", "TMPDIR=/tmp", "HTTPS_PROXY=http://proxy:3128",
		"GIT_SSH_COMMAND=ssh", "XDG_CONFIG_HOME=/home/user/.config",
	} {
		if !strings.Contains(joined, keep) {
			t.Errorf("a child needs %q; env was:\n%s", keep, joined)
		}
	}
}

func TestAnExplicitAllowOverridesTheDefaultDeny(t *testing.T) {
	// A deployment that genuinely needs a variable in a child says so by name.
	parent := []string{"PATH=/usr/bin", "OPENCODE_API_KEY=key-value"}
	got := strings.Join(BuildFrom(parent, Options{Allow: []string{"OPENCODE_API_KEY"}}), "\n")
	if !strings.Contains(got, "OPENCODE_API_KEY=key-value") {
		t.Fatalf("an explicitly allowed variable must pass through; env was:\n%s", got)
	}
	// And only that one.
	if strings.Contains(got, "PATH=/usr/bin\nPATH") {
		t.Fatal("unexpected duplication")
	}
}

func TestAnExplicitAllowIsStillRefusedForTheHardDeniedNames(t *testing.T) {
	// Some names are the harness's own credentials. Handing them to an arbitrary
	// child is the exact hole, so they need the Extra path, not Allow.
	parent := []string{"PRUMO_MODEL_API_KEY=key-value"}
	if got := strings.Join(BuildFrom(parent, Options{Allow: []string{"PRUMO_MODEL_API_KEY"}}), "\n"); strings.Contains(got, "key-value") {
		t.Fatalf("a harness credential must not be passable by Allow; env was:\n%s", got)
	}
}

func TestExtraIsSetOnTheChild(t *testing.T) {
	parent := []string{"PATH=/usr/bin"}
	got := strings.Join(BuildFrom(parent, Options{Extra: map[string]string{
		"PRUMO_CHILD": "yes", "PATH": "/opt/bin",
	}}), "\n")
	if !strings.Contains(got, "PRUMO_CHILD=yes") {
		t.Errorf("Extra must reach the child; env was:\n%s", got)
	}
	if !strings.Contains(got, "PATH=/opt/bin") {
		t.Errorf("Extra must be able to override an inherited value; env was:\n%s", got)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	// A stable environment makes a child's behaviour reproducible and a test
	// comparable.
	parent := []string{"TZ=UTC", "PATH=/usr/bin", "HOME=/home/u", "LANG=C"}
	first := strings.Join(BuildFrom(parent, Options{}), "\n")
	second := strings.Join(BuildFrom(parent, Options{}), "\n")
	if first != second {
		t.Fatalf("the environment must be stable:\n%s\n---\n%s", first, second)
	}
	if !strings.HasPrefix(first, "HOME=") && !strings.HasPrefix(first, "LANG=") {
		t.Fatalf("the environment must be sorted: %s", first)
	}
}

func TestMalformedEntriesAreIgnored(t *testing.T) {
	got := strings.Join(BuildFrom([]string{"", "=novalue", "NOEQUALS", "PATH=/usr/bin"}, Options{}), "\n")
	if strings.Contains(got, "NOEQUALS") || strings.Contains(got, "novalue") {
		t.Fatalf("an entry with no name or no separator must be dropped; env was:\n%s", got)
	}
	if !strings.Contains(got, "PATH=/usr/bin") {
		t.Fatalf("the valid entry must survive; env was:\n%s", got)
	}
}

func TestRedactRemovesSecretValuesFromChildOutput(t *testing.T) {
	text := "error: request failed with key sk-live-1234567890abcdef"
	got := Redact(text, []string{"sk-live-1234567890abcdef"})
	if strings.Contains(got, "sk-live-1234567890abcdef") {
		t.Fatalf("a secret in child output must be scrubbed, got %q", got)
	}
	if !strings.Contains(got, RedactedPrefix) {
		t.Fatalf("the redaction must be visible in place, got %q", got)
	}
}

func TestRedactLeavesShortValuesAlone(t *testing.T) {
	// A two-character secret would match ordinary text and corrupt the output
	// for no gain.
	text := "an error occurred at id=ab"
	if got := Redact(text, []string{"ab"}); got != text {
		t.Fatalf("a short value must not be redacted, got %q", got)
	}
}

func TestRedactHandlesNoSecrets(t *testing.T) {
	text := "ordinary output"
	if got := Redact(text, nil); got != text {
		t.Fatalf("redaction with no secrets must be a no-op, got %q", got)
	}
}
