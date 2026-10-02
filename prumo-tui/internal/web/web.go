package web

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	scriptRegex = regexp.MustCompile(`(?is)<script.*?</script>`)
	styleRegex  = regexp.MustCompile(`(?is)<style.*?</style>`)
	navRegex    = regexp.MustCompile(`(?is)<nav.*?</nav>`)
	footerRegex = regexp.MustCompile(`(?is)<footer.*?</footer>`)
	headerRegex = regexp.MustCompile(`(?is)<header.*?</header>`)
	tagRegex    = regexp.MustCompile(`(?s)<[^>]+>`)
	spacesRegex = regexp.MustCompile(`[ \t]+`)
	linesRegex  = regexp.MustCompile(`\n{3,}`)

	h1Regex = regexp.MustCompile(`(?i)<h1[^>]*>(.*?)</h1>`)
	h2Regex = regexp.MustCompile(`(?i)<h2[^>]*>(.*?)</h2>`)
	h3Regex = regexp.MustCompile(`(?i)<h3[^>]*>(.*?)</h3>`)
	liRegex = regexp.MustCompile(`(?i)<li[^>]*>(.*?)</li>`)
	pRegex  = regexp.MustCompile(`(?i)<p[^>]*>(.*?)</p>`)
)

const (
	maxResponseBodyBytes = 512 * 1024
	maxDistilledLength   = 6000
)

// FetchAndDistill retrieves a web page and distills it into readable Markdown,
// stripping menus, scripts, stylesheets, and excessive markup to conserve context.
func FetchAndDistill(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid URL %q: scheme must be http or https", rawURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "Prumo-TUI/0.7.0 (WebDistiller; +https://github.com/raillen/prumo)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.8")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP error %d %s", resp.StatusCode, resp.Status)
	}

	limitedReader := io.LimitReader(resp.Body, maxResponseBodyBytes)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return DistillHTML(string(bodyBytes)), nil
}

// DistillHTML converts raw HTML text into clean, token-efficient Markdown.
func DistillHTML(rawHTML string) string {
	// Strip heavy non-content sections
	text := scriptRegex.ReplaceAllString(rawHTML, "")
	text = styleRegex.ReplaceAllString(text, "")
	text = navRegex.ReplaceAllString(text, "")
	text = footerRegex.ReplaceAllString(text, "")
	text = headerRegex.ReplaceAllString(text, "")

	// Convert basic structural elements to Markdown
	text = h1Regex.ReplaceAllString(text, "\n# $1\n")
	text = h2Regex.ReplaceAllString(text, "\n## $1\n")
	text = h3Regex.ReplaceAllString(text, "\n### $1\n")
	text = liRegex.ReplaceAllString(text, "\n- $1")
	text = pRegex.ReplaceAllString(text, "\n$1\n")

	// Strip remaining HTML tags
	text = tagRegex.ReplaceAllString(text, " ")

	// Unescape HTML entities
	text = html.UnescapeString(text)

	// Clean up whitespace
	text = spacesRegex.ReplaceAllString(text, " ")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var cleanLines []string
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}
	cleaned := strings.Join(cleanLines, "\n\n")
	cleaned = linesRegex.ReplaceAllString(cleaned, "\n\n")

	if len(cleaned) > maxDistilledLength {
		cleaned = cleaned[:maxDistilledLength] + "\n\n... [content truncated to conserve token budget]"
	}

	return cleaned
}
