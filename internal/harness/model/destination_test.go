package model

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"strings"
	"testing"
)

// The provider's base URL is a request field. Without a destination check the
// harness sends the conversation and the API key wherever the caller names —
// the cloud metadata endpoint, a loopback service, the office LAN. These are
// the destinations that must never be reachable by default.

func TestValidateDestinationURLRejectsCloudMetadata(t *testing.T) {
	for _, raw := range []string{
		"https://169.254.169.254/latest/meta-data/iam/security-credentials/",
		"http://169.254.169.254/",
		"https://[fd00:ec2::254]/latest/meta-data/",
	} {
		if err := ValidateDestinationURL(raw, DefaultDestinationPolicy()); err == nil {
			t.Errorf("%s is a cloud metadata endpoint and must be refused", raw)
		}
	}
}

func TestValidateDestinationURLRejectsPrivateAndLoopback(t *testing.T) {
	for _, raw := range []string{
		"https://10.0.0.5/v1",
		"https://192.168.1.1/v1",
		"https://172.16.0.1/v1",
		"https://127.0.0.1:11434/v1",
		"https://localhost:8080/v1",
		"https://[::1]:8080/v1",
		"https://169.254.10.1/v1",
	} {
		if err := ValidateDestinationURL(raw, DefaultDestinationPolicy()); err == nil {
			t.Errorf("%s is internal and must be refused by default", raw)
		}
	}
}

func TestValidateDestinationURLRejectsNonHTTPSchemes(t *testing.T) {
	for _, raw := range []string{
		"file:///etc/passwd",
		"gopher://evil/",
		"ftp://evil/",
		"data:text/plain,hi",
		"http://api.example.com/v1",
	} {
		if err := ValidateDestinationURL(raw, DefaultDestinationPolicy()); err == nil {
			t.Errorf("%s must be refused", raw)
		}
	}
}

func TestValidateDestinationURLRejectsMalformed(t *testing.T) {
	for _, raw := range []string{"", "   ", "https://", "not a url at all", "://missing-scheme"} {
		if err := ValidateDestinationURL(raw, DefaultDestinationPolicy()); err == nil {
			t.Errorf("%q must be refused", raw)
		}
	}
}

func TestValidateDestinationURLAllowsPublicHTTPS(t *testing.T) {
	for _, raw := range []string{
		"https://api.openai.com/v1",
		"https://openrouter.ai/api/v1",
		"https://generativelanguage.googleapis.com/v1beta",
	} {
		if err := ValidateDestinationURL(raw, DefaultDestinationPolicy()); err != nil {
			t.Errorf("%s is a public https endpoint and must be allowed: %v", raw, err)
		}
	}
}

func TestValidateDestinationURLHonoursHostAllowlist(t *testing.T) {
	policy := DestinationPolicy{AllowedHosts: []string{"gateway.internal.example"}}
	if err := ValidateDestinationURL("https://gateway.internal.example/v1", policy); err != nil {
		t.Fatalf("a host in the allowlist must be allowed: %v", err)
	}
	if err := ValidateDestinationURL("https://elsewhere.example/v1", policy); err == nil {
		t.Fatal("a host outside the allowlist must be refused")
	}
	// The allowlist is not a bypass: a listed host resolving privately is still
	// caught, because the dialer checks the address the name resolves to.
	if err := policy.checkAddr(mustAddr(t, "10.1.2.3")); err == nil {
		t.Fatal("the allowlist must not permit a private address")
	}
}

func TestLocalDevelopmentPolicyAllowsLoopbackOverPlainHTTP(t *testing.T) {
	// A human running llama.cpp locally needs this, and it must be opt-in.
	policy := LocalDevelopmentDestinationPolicy()
	if err := ValidateDestinationURL("http://127.0.0.1:11434/v1", policy); err != nil {
		t.Fatalf("local development policy must allow a loopback gateway: %v", err)
	}
	// Even opted in, it does not open the rest of the network.
	if err := ValidateDestinationURL("https://169.254.169.254/", policy); err == nil {
		t.Fatal("the local policy must still refuse the metadata endpoint")
	}
	if err := ValidateDestinationURL("https://10.0.0.5/v1", policy); err == nil {
		t.Fatal("the local policy must still refuse private networks")
	}
}

func TestMetadataPolicyIsSeparateFromLoopbackPolicy(t *testing.T) {
	// Allowing loopback must not be the back door to the metadata endpoint.
	policy := DestinationPolicy{AllowLoopback: true}
	if err := policy.checkAddr(mustAddr(t, "169.254.169.254")); err == nil {
		t.Fatal("AllowLoopback must not permit the metadata address")
	}
}

// The dialer is where the check has to live. A hostname that resolves to a
// private address is the interesting case, and it is why the test resolves a
// real name rather than using a literal.

func TestDestinationClientRefusesAHostResolvingToLoopback(t *testing.T) {
	if testing.Short() {
		t.Skip("dns-dependent")
	}
	client := NewDestinationHTTPClient(DefaultDestinationPolicy())
	_, err := client.Get("http://localhost:" + unusedLocalPort(t) + "/")
	if err == nil {
		t.Fatal("a host resolving to loopback must be refused by the default policy")
	}
	if !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("the refusal must name the reason, got %v", err)
	}
}

func TestDestinationClientRefusesTheMetadataEndpointByName(t *testing.T) {
	if testing.Short() {
		t.Skip("dns-dependent")
	}
	// AWS answers this on every instance; on a laptop it usually NXDOMAINs, and
	// either outcome must not produce a request.
	client := NewDestinationHTTPClient(DefaultDestinationPolicy())
	_, err := client.Get("http://metadata.google.internal/computeMetadata/v1/")
	if err == nil {
		t.Fatal("a metadata hostname must not produce a request")
	}
}

func TestDestinationClientRefusesAPlainHTTPProvider(t *testing.T) {
	client := NewDestinationHTTPClient(DefaultDestinationPolicy())
	_, err := client.Get("http://api.example.com/v1/models")
	if err == nil {
		t.Fatal("plain http to a provider must be refused by the default policy")
	}
}

func TestDestinationClientRejectsANonHTTPURLBeforeDialing(t *testing.T) {
	client := NewDestinationHTTPClient(DefaultDestinationPolicy())
	if _, err := client.Get("file:///etc/passwd"); err == nil {
		t.Fatal("a file URL must be refused")
	}
}

func TestLocalPolicyClientReachesALoopbackServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	client := NewDestinationHTTPClient(LocalDevelopmentDestinationPolicy())
	resp, err := client.Get(srv.URL)
	if err != nil {
		t.Fatalf("the local policy must reach a loopback server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status %d", resp.StatusCode)
	}
}

func TestFactoryRefusesAMetadataBaseURL(t *testing.T) {
	// The check belongs at the factory too, so a caller learns before any
	// conversation is assembled.
	if _, err := ForName("openai-compat", "https://169.254.169.254/v1", "key", "m"); err == nil {
		t.Fatal("the factory must refuse a metadata base URL")
	}
	if _, err := ForName("anthropic", "https://10.0.0.5/v1", "key", "m"); err == nil {
		t.Fatal("the factory must refuse a private base URL")
	}
}

func TestFactoryAcceptsAPublicBaseURL(t *testing.T) {
	p, err := ForName("openai-compat", "https://openrouter.ai/api/v1", "key", "m")
	if err != nil {
		t.Fatalf("a public https endpoint must be accepted: %v", err)
	}
	if p == nil {
		t.Fatal("a provider must be returned")
	}
}

func mustAddr(t *testing.T, raw string) netip.Addr {
	t.Helper()
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		t.Fatalf("parse %s: %v", raw, err)
	}
	return addr
}

// unusedLocalPort returns a port nothing is listening on, so the dialer fails on
// the address check rather than on a refused connection.
func unusedLocalPort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return strconv.Itoa(port)
}
