package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDistillHTML(t *testing.T) {
	rawHTML := `
<!DOCTYPE html>
<html>
<head>
	<title>Test Page</title>
	<style>body { color: red; }</style>
	<script>console.log("secret");</script>
</head>
<body>
	<nav><a href="/">Home</a></nav>
	<h1>Welcome to Prumo</h1>
	<p>Prumo is a <strong>cognitive</strong> framework &amp; orchestrator.</p>
	<h2>Features</h2>
	<ul>
		<li>Lean Progressive Context</li>
		<li>Native TUI &amp; IDE</li>
	</ul>
	<footer>Copyright 2026</footer>
</body>
</html>
`
	distilled := DistillHTML(rawHTML)

	if strings.Contains(distilled, "console.log") {
		t.Fatal("expected script tag to be stripped")
	}
	if strings.Contains(distilled, "body { color") {
		t.Fatal("expected style tag to be stripped")
	}
	if strings.Contains(distilled, "Home") {
		t.Fatal("expected nav tag to be stripped")
	}
	if strings.Contains(distilled, "Copyright 2026") {
		t.Fatal("expected footer tag to be stripped")
	}
	if !strings.Contains(distilled, "# Welcome to Prumo") {
		t.Fatalf("expected # Welcome to Prumo heading in output, got: %s", distilled)
	}
	if !strings.Contains(distilled, "cognitive framework & orchestrator") {
		t.Fatalf("expected unescaped entity in output, got: %s", distilled)
	}
	if !strings.Contains(distilled, "- Lean Progressive Context") {
		t.Fatalf("expected markdown list item, got: %s", distilled)
	}
}

func TestFetchAndDistill(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
			<h1>Prumo Server Test</h1>
			<p>Live HTTP fetch test passed.</p>
		`))
	}))
	defer ts.Close()

	ctx := context.Background()
	result, err := FetchAndDistill(ctx, ts.URL)
	if err != nil {
		t.Fatalf("unexpected error fetching test server: %v", err)
	}

	if !strings.Contains(result, "# Prumo Server Test") {
		t.Fatalf("expected heading in result, got: %s", result)
	}
	if !strings.Contains(result, "Live HTTP fetch test passed.") {
		t.Fatalf("expected paragraph text in result, got: %s", result)
	}
}

func TestFetchAndDistillInvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := FetchAndDistill(ctx, "ftp://invalid-scheme.com")
	if err == nil {
		t.Fatal("expected error for non-http(s) scheme")
	}
}
